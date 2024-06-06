package pixi

import (
	"io"
)

type CacheTile struct {
	Dirty bool
	Data  []byte
}

type CacheDataset struct {
	DataSet
	TileCache  map[uint]*CacheTile
	MaxInCache uint
	Backing    io.ReadWriteSeeker
}

func NewCacheDataset(d DataSet, backing io.ReadWriteSeeker, maxInCache uint, offset int64) (*CacheDataset, error) {
	cacheSet := &CacheDataset{DataSet: d}
	cacheSet.Backing = backing
	cacheSet.MaxInCache = maxInCache
	cacheSet.TileCache = make(map[uint]*CacheTile, maxInCache)
	cacheSet.Offset = offset

	// populate backing data store with empty data
	tileCount := cacheSet.Tiles()
	if cacheSet.Separated {
		tileCount *= len(cacheSet.Fields)
	}
	_, err := backing.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 0)
	for i := 0; i < tileCount; i++ {
		tileSize := cacheSet.TileSize(i)
		if tileSize != int64(len(buf)) {
			buf = make([]byte, tileSize)
		}
		_, err := backing.Write(buf)
		if err != nil {
			return nil, err
		}
	}

	return cacheSet, nil
}

func ReadCached(r io.ReadWriteSeeker, ds DataSet) (CacheDataset, error) {
	cached := CacheDataset{DataSet: ds, TileCache: make(map[uint]*CacheTile), Backing: r}
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
				loaded, err := d.loadTile(tileIndex)
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
				loaded, err := d.loadTile(tileIndex)
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

// Load a tile from the cache or disk, if it's not in memory.
//
// This function is responsible for loading a tile into memory if it's not already there.
// It does this by first checking if the tile exists in the cache, and if so, returns it directly.
// If not, it reads the tile from disk and caches it before returning.
func (d *CacheDataset) loadTile(tileIndex uint) (*CacheTile, error) {
	for len(d.TileCache) >= int(d.MaxInCache) {
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
	tileOffset := d.Offset
	if d.Separated {
		for iterInd := tileIndex; iterInd > 0; iterInd = iterInd - uint(d.Tiles()) {
			tileOffset += d.TileSize(int(iterInd)) * int64(d.Tiles())
		}
	}
	tileOffset += d.TileSize(int(tileIndex)) * int64(tileIndex)
	d.Backing.Seek(tileOffset, io.SeekStart)

	buf := make([]byte, d.TileSize(int(tileIndex)))
	_, err := d.Backing.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// This function writes a tile from memory back to the underlying storage.
// The offset of the tile in the storage is determined by the `tileIndex`.
func (d *CacheDataset) writeTile(data []byte, tileIndex uint) error {
	tileOffset := d.Offset
	if d.Separated {
		for iterInd := tileIndex; iterInd > 0; iterInd = iterInd - uint(d.Tiles()) {
			tileOffset += d.TileSize(int(iterInd)) * int64(d.Tiles())
		}
	}
	tileOffset += d.TileSize(int(tileIndex)) * int64(tileIndex)
	d.Backing.Seek(tileOffset, io.SeekStart)

	_, err := d.Backing.Write(data)
	if err != nil {
		return err
	}
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
