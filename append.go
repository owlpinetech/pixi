package pixi

import (
	"bytes"
	"compress/flate"
	"io"
)

type AppendTile struct {
	Dirty bool
	Data  []byte
}

type AppendDataset struct {
	DataSet
	WritingTileIndex uint
	WritingTile      AppendTile
	ReadCache        map[uint]*AppendTile
	MaxInCache       uint
	Backing          io.ReadWriteSeeker
}

// Creates a new append dataset. It initializes the internal data structures and sets up the backing store.
func NewAppendDataset(d DataSet, backing io.ReadWriteSeeker, maxInCache uint, offset int64) (*AppendDataset, error) {
	appendSet := &AppendDataset{DataSet: d}
	appendSet.Backing = backing
	appendSet.ReadCache = make(map[uint]*AppendTile)
	appendSet.MaxInCache = maxInCache
	appendSet.Offset = offset
	appendSet.WritingTileIndex = 0
	appendSet.WritingTile = AppendTile{Data: make([]byte, appendSet.TileSize(0))}
	appendSet.ReadCache[appendSet.WritingTileIndex] = &appendSet.WritingTile

	diskTileCount := appendSet.Tiles()
	if appendSet.Separated {
		diskTileCount *= len(appendSet.Fields)
	}
	appendSet.TileBytes = make([]int64, diskTileCount)

	return appendSet, nil
}

func ReadAppend(r io.ReadWriteSeeker, ds DataSet) (AppendDataset, error) {
	appended := AppendDataset{DataSet: ds, ReadCache: make(map[uint]*AppendTile), Backing: r}
	return appended, nil
}

func (d *AppendDataset) GetSample(dimIndices []uint) ([]any, error) {
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
			var cached *AppendTile
			if tile, ok := d.ReadCache[fieldTile]; ok {
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
		var cached *AppendTile
		if tile, ok := d.ReadCache[tileIndex]; ok {
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

func (d *AppendDataset) GetSampleField(dimIndices []uint, fieldId uint) (any, error) {
	if len(d.Dimensions) != len(dimIndices) {
		return nil, DimensionsError{len(d.Dimensions), len(dimIndices)}
	}

	tileIndex, inTileIndex := d.dimIndicesToTileIndices(dimIndices)
	if d.Separated {
		tileIndex += uint(d.Tiles()) * uint(fieldId)
		inTileIndex *= uint(d.Fields[fieldId].Size())
	} else {
		inTileIndex *= uint(d.SampleSize())
	}

	// TODO: locking for safe concurrent access
	var cached *AppendTile
	if tile, ok := d.ReadCache[tileIndex]; ok {
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

func (d *AppendDataset) SetSample(dimIndices []uint, sample []any) error {
	if len(d.Dimensions) != len(dimIndices) {
		return DimensionsError{len(d.Dimensions), len(dimIndices)}
	}
	if d.Separated {
		return UnsupportedError("cannot write a full sample in a separated append-only data set")
	}

	tileIndex, inTileIndex := d.dimIndicesToTileIndices(dimIndices)
	// check if we need to move to the next tile or if we're out of range
	if tileIndex != d.WritingTileIndex {
		if tileIndex == d.WritingTileIndex+1 {
			err := d.writeCompressTile(d.WritingTile.Data, d.WritingTileIndex)
			if err != nil {
				return err
			}
			d.WritingTileIndex += 1
			d.WritingTile = AppendTile{Data: make([]byte, d.TileSize(int(d.WritingTileIndex)))}
			err = d.addTileToCache(d.WritingTileIndex, d.WritingTile)
			if err != nil {
				return err
			}
		} else {
			return RangeError{Specified: int(tileIndex), ValidMin: int(d.WritingTileIndex), ValidMax: int(d.WritingTileIndex)}
		}
	}

	inTileIndex *= uint(d.SampleSize())
	for fieldId, field := range d.Fields {
		field.Write(d.WritingTile.Data[inTileIndex:], sample[fieldId])
		inTileIndex += uint(field.Size())
	}

	return nil
}

func (d *AppendDataset) SetSampleField(dimIndices []uint, fieldId uint, fieldVal any) error {
	if len(d.Dimensions) != len(dimIndices) {
		return DimensionsError{len(d.Dimensions), len(dimIndices)}
	}

	tileIndex, inTileIndex := d.dimIndicesToTileIndices(dimIndices)
	if d.Separated {
		tileIndex += uint(d.Tiles()) * uint(fieldId)
		inTileIndex *= uint(d.Fields[fieldId].Size())
	} else {
		inTileIndex *= uint(d.SampleSize())
	}

	// check if we need to move to the next tile or if we're out of range
	if tileIndex != d.WritingTileIndex {
		if tileIndex == d.WritingTileIndex+1 {
			err := d.writeCompressTile(d.WritingTile.Data, d.WritingTileIndex)
			if err != nil {
				return err
			}
			d.WritingTileIndex = tileIndex
			d.WritingTile = AppendTile{Data: make([]byte, d.TileSize(int(d.WritingTileIndex)))}
			err = d.addTileToCache(d.WritingTileIndex, d.WritingTile)
			if err != nil {
				return err
			}
		} else {
			return RangeError{Specified: int(tileIndex), ValidMin: int(d.WritingTileIndex), ValidMax: int(d.WritingTileIndex)}
		}
	}

	d.Fields[fieldId].Write(d.WritingTile.Data[inTileIndex:], fieldVal)
	return nil
}

func (d *AppendDataset) addTileToCache(tileIndex uint, tile AppendTile) error {
	for len(d.ReadCache) >= int(d.MaxInCache) {
		err := d.evict()
		if err != nil {
			return err
		}
	}

	d.ReadCache[tileIndex] = &tile
	return nil
}

// Load a tile from the cache or disk, if it's not in memory.
//
// This function is responsible for loading a tile into memory if it's not already there.
// It does this by first checking if the tile exists in the cache, and if so, returns it directly.
// If not, it reads the tile from disk and caches it before returning.
func (d *AppendDataset) loadTile(tileIndex uint) (*AppendTile, error) {
	read, err := d.readTile(tileIndex)
	if err != nil {
		return nil, err
	}
	err = d.addTileToCache(tileIndex, AppendTile{Data: read})
	return d.ReadCache[tileIndex], err
}

// Evicts the oldest cached tile and writes its data to the underlying storage.
// This method is used when the cache exceeds its maximum size.
// It ensures that all changes made by this dataset are persisted.
// Return an error if there was an issue with persisting or evicting a tile, nil otherwise
func (d *AppendDataset) evict() error {
	if len(d.ReadCache) == 0 {
		return nil
	}
	var first uint
	for k := range d.ReadCache {
		first = k
		break
	}

	delete(d.ReadCache, first)
	return nil
}

// Read a tile from the backing storage.
// This function reads a tile from the underlying storage and returns its data as a byte slice.
// The offset of the tile in the storage is determined by the `tileIndex`.
func (d *AppendDataset) readTile(tileIndex uint) ([]byte, error) {
	if tileIndex > d.WritingTileIndex {
		return nil, RangeError{Specified: int(tileIndex), ValidMin: 0, ValidMax: int(d.WritingTileIndex)}
	}
	d.Backing.Seek(d.TileOffset(int(tileIndex)), io.SeekStart)

	uncompressedLen := d.TileSize(int(tileIndex))
	buf := make([]byte, uncompressedLen)

	switch d.Compression {
	case CompressionNone:
		_, err := d.Backing.Read(buf)
		if err != nil {
			return nil, err
		}
		return buf, nil
	case CompressionGzip:
		gzRdr := flate.NewReader(d.Backing)
		defer gzRdr.Close()
		_, err := gzRdr.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
	}

	return buf, nil
}

// This function takes in a byte slice and a tile index as input, and writes the contents of the slice to the backing storage at the specified tile offset.
// The function is responsible for handling both uncompressed and gzip-compressed data.
// If there was an issue with writing the tile, this function will return an error. Otherwise, it returns nil.
func (d *AppendDataset) writeCompressTile(data []byte, tileIndex uint) error {
	tileOffset := d.Offset
	for _, bytes := range d.TileBytes[:tileIndex] {
		tileOffset += bytes
	}
	d.Backing.Seek(tileOffset, io.SeekStart)

	tileSize := 0
	switch d.Compression {
	case CompressionNone:
		written, err := d.Backing.Write(data)
		if err != nil {
			return err
		}
		tileSize = written
	case CompressionGzip:
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
		_, err = d.Backing.Write(buf.Bytes())
		if err != nil {
			return err
		}
		tileSize = len(buf.Bytes())
	}

	// make sure to update the byte counts for this tile
	d.TileBytes[tileIndex] = int64(tileSize)
	return nil
}

// This function takes a slice of dimension indices and converts them into a tile index.
func (d *AppendDataset) dimIndicesToTileIndices(dimIndices []uint) (tileIndex uint, inTileIndex uint) {
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
