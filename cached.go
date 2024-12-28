package pixi

import (
	"bytes"
	"compress/flate"
	"io"
)

type CacheTile struct {
	Dirty bool
	Data  []byte
}

type CacheDataset struct {
	*DiskLayer
	TileCache  map[uint]*CacheTile
	MaxInCache uint
	Backing    io.ReadWriteSeeker
}

func ReadCached(r io.ReadWriteSeeker, dl *DiskLayer, maxInCache uint) (*CacheDataset, error) {
	if dl.Compression != CompressionNone {
		return nil, UnsupportedError("CacheDataset type currently does not supported compressed data sets")
	}
	cached := &CacheDataset{DiskLayer: dl, TileCache: make(map[uint]*CacheTile), Backing: r, MaxInCache: maxInCache}
	return cached, nil
}

func (d *CacheDataset) GetSample(dimIndices []uint) ([]any, error) {
	if len(d.Dimensions) != len(dimIndices) {
		return nil, DimensionsError{len(d.Dimensions), len(dimIndices)}
	}

	tileIndex, inTileIndex := d.dimIndicesToTileIndices(dimIndices)

	sample := make([]any, len(d.Fields))

	if d.Separated {
		for fieldId, field := range d.Fields {
			fieldTile := tileIndex + uint(d.Tiles())*uint(fieldId)
			fieldOffset := inTileIndex * uint(field.Size())

			// TODO: locking for safe concurrent access
			var cached *CacheTile
			if tile, ok := d.TileCache[fieldTile]; ok {
				cached = tile
			} else {
				loaded, err := d.loadTile(fieldTile)
				if err != nil {
					return nil, err
				} else {
					cached = loaded
				}
			}

			fieldVal := field.Read(cached.Data[fieldOffset:])
			sample[fieldId] = fieldVal
		}
	} else {
		fieldOffset := inTileIndex * uint(d.SampleSize())

		// TODO: locking for safe concurrent access
		var cached *CacheTile
		if tile, ok := d.TileCache[tileIndex]; ok {
			cached = tile
		} else {
			loaded, err := d.loadTile(tileIndex)
			if err != nil {
				return nil, err
			} else {
				cached = loaded
			}
		}

		for fieldId, field := range d.Fields {
			fieldVal := field.Read(cached.Data[fieldOffset:])
			sample[fieldId] = fieldVal

			fieldOffset += uint(field.Size())
		}
	}

	return sample, nil
}

func (d *CacheDataset) GetSampleField(dimIndices []uint, fieldId uint) (any, error) {
	if len(d.Dimensions) != len(dimIndices) {
		return nil, DimensionsError{len(d.Dimensions), len(dimIndices)}
	}

	tileIndex, inTileIndex := d.dimIndicesToTileIndices(dimIndices)
	if d.Separated {
		tileIndex += uint(d.Tiles()) * uint(fieldId)
		inTileIndex *= uint(d.Fields[fieldId].Size())
	} else {
		inTileIndex *= uint(d.SampleSize())
		for _, field := range d.Fields[:fieldId] {
			inTileIndex += uint(field.Size())
		}
	}

	// TODO: locking for safe concurrent access
	var cached *CacheTile
	if tile, ok := d.TileCache[tileIndex]; ok {
		cached = tile
	} else {
		loaded, err := d.loadTile(tileIndex)
		if err != nil {
			return nil, err
		} else {
			cached = loaded
		}
	}

	return d.Fields[fieldId].Read(cached.Data[inTileIndex:]), nil
}

func (d *CacheDataset) SetSample(dimIndices []uint, sample []any) error {
	if len(d.Dimensions) != len(dimIndices) {
		return DimensionsError{len(d.Dimensions), len(dimIndices)}
	}

	tileIndex, inTileIndex := d.dimIndicesToTileIndices(dimIndices)

	if d.Separated {
		for fieldId, field := range d.Fields {
			fieldTile := tileIndex + uint(d.Tiles())*uint(fieldId)
			fieldOffset := inTileIndex * uint(field.Size())

			// TODO: locking for safe concurrent access
			var cached *CacheTile
			if tile, ok := d.TileCache[fieldTile]; ok {
				cached = tile
			} else {
				loaded, err := d.loadTile(fieldTile)
				if err != nil {
					return err
				} else {
					cached = loaded
				}
			}

			field.Write(cached.Data[fieldOffset:], sample[fieldId])
			cached.Dirty = true
		}
	} else {
		fieldOffset := inTileIndex * uint(d.SampleSize())

		// TODO: locking for safe concurrent access
		var cached *CacheTile
		if tile, ok := d.TileCache[tileIndex]; ok {
			cached = tile
		} else {
			loaded, err := d.loadTile(tileIndex)
			if err != nil {
				return err
			} else {
				cached = loaded
			}
		}

		for fieldId, field := range d.Fields {
			field.Write(cached.Data[fieldOffset:], sample[fieldId])
			fieldOffset += uint(field.Size())
		}
		cached.Dirty = true
	}

	return nil
}

func (d *CacheDataset) SetSampleField(dimIndices []uint, fieldId uint, fieldVal any) error {
	if len(d.Dimensions) != len(dimIndices) {
		return DimensionsError{len(d.Dimensions), len(dimIndices)}
	}

	tileIndex, inTileIndex := d.dimIndicesToTileIndices(dimIndices)
	if d.Separated {
		tileIndex += uint(d.Tiles()) * uint(fieldId)
		inTileIndex *= uint(d.Fields[fieldId].Size())
	} else {
		inTileIndex *= uint(d.SampleSize())
		for _, field := range d.Fields[:fieldId] {
			inTileIndex += uint(field.Size())
		}
	}

	// TODO: locking for safe concurrent access
	var cached *CacheTile
	if tile, ok := d.TileCache[tileIndex]; ok {
		cached = tile
	} else {
		loaded, err := d.loadTile(tileIndex)
		if err != nil {
			return err
		} else {
			cached = loaded
		}
	}

	cached.Dirty = true
	d.Fields[fieldId].Write(cached.Data[inTileIndex:], fieldVal)
	return nil
}

func (d *CacheDataset) Finalize(pix *Pixi) error {
	for tileInd, tile := range d.TileCache {
		if tile.Dirty {
			err := d.writeTile(tile.Data, tileInd)
			if err != nil {
				return err
			}
		}
	}

	_, err := d.Backing.Seek(pix.LayerOffset(d.DiskLayer), io.SeekStart)
	if err != nil {
		return err
	}
	err = WriteLayerHeader(d.Backing, *d.DiskLayer)
	if err != nil {
		return err
	}

	return nil
}

// Load a tile from the cache or disk, if it's not in memory.
//
// This function is responsible for loading a tile into memory if it's not already there.
// It does this by first checking if the tile exists in the cache, and if so, returns it directly.
// If not, it reads the tile from disk and caches it before returning.
func (d *CacheDataset) loadTile(tileIndex uint) (*CacheTile, error) {
	if len(d.TileCache) >= int(d.MaxInCache) {
		err := d.evict()
		if err != nil {
			return nil, err
		}
	}

	read, err := d.readTile(tileIndex)
	d.TileCache[tileIndex] = &CacheTile{Data: read}
	return d.TileCache[tileIndex], err
}

// Evicts the oldest cached tile and writes its data to the underlying storage.
// This method is used when the cache exceeds its maximum size.
// It ensures that all changes made by this dataset are persisted.
// Return an error if there was an issue with persisting or evicting a tile, nil otherwise
func (d *CacheDataset) evict() error {
	if len(d.TileCache) <= 0 {
		return nil
	}
	var first uint
	for k := range d.TileCache {
		first = k
		break
	}

	if d.TileCache[first].Dirty {
		err := d.writeTile(d.TileCache[first].Data, first)
		if err != nil {
			return err
		}
	}

	delete(d.TileCache, first)
	return nil
}

// Read a tile from the backing storage.
// This function reads a tile from the underlying storage and returns its data as a byte slice.
// The offset of the tile in the storage is determined by the `tileIndex`.
func (d *CacheDataset) readTile(tileIndex uint) ([]byte, error) {
	d.Backing.Seek(d.TileOffsets[int(tileIndex)], io.SeekStart)

	uncompressedLen := d.DiskTileSize(int(tileIndex))

	switch d.Compression {
	case CompressionNone:
		buf := make([]byte, uncompressedLen)
		_, err := d.Backing.Read(buf)
		if err != nil {
			return nil, err
		}
		return buf, nil
	case CompressionFlate:
		buf := make([]byte, 0, uncompressedLen)
		bufRd := bytes.NewBuffer(buf)
		gzRdr := flate.NewReader(d.Backing)
		defer gzRdr.Close()
		_, err := io.Copy(bufRd, gzRdr)
		if err != nil {
			return nil, err
		}
		return bufRd.Bytes(), nil
	}

	return nil, UnsupportedError("unknown compression type")
}

// This function writes a tile from memory back to the underlying storage.
// The offset of the tile in the storage is determined by the `tileIndex`.
func (d *CacheDataset) writeTile(data []byte, tileIndex uint) error {
	offset := d.TileOffsets[int(tileIndex)]
	d.Backing.Seek(offset, io.SeekStart)

	tileSize := 0
	switch d.Compression {
	case CompressionNone:
		written, err := d.Backing.Write(data)
		if err != nil {
			return err
		}
		tileSize = written
	case CompressionFlate:
		buf := new(bytes.Buffer)
		gzWtr, err := flate.NewWriter(buf, flate.BestCompression)
		if err != nil {
			return err
		}
		_, err = gzWtr.Write(data)
		if err != nil {
			gzWtr.Close()
			return err
		}
		gzWtr.Close()
		tileSize = buf.Len()
		_, err = io.Copy(d.Backing, buf)
		if err != nil {
			return err
		}
	}

	// make sure to update the byte counts for this tile
	d.TileBytes[tileIndex] = int64(tileSize)
	return nil
}

func (d *CacheDataset) dimIndicesToTileIndices(dimIndices []uint) (tileIndex uint, inTileIndex uint) {
	tileIndex = uint(0)
	inTileIndex = uint(0)
	tileMul := uint(1)
	inTileMul := uint(1)
	for dInd, index := range dimIndices {
		tileIndex += (index / uint(d.Dimensions[dInd].TileSize)) * tileMul
		inTileIndex += (index % uint(d.Dimensions[dInd].TileSize)) * inTileMul
		tileMul *= uint(d.Dimensions[dInd].Tiles())
		inTileMul *= uint(d.Dimensions[dInd].TileSize)
	}
	return
}
