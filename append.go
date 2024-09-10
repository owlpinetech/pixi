package pixi

/*
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
	Pixi
	WritingTileIndex uint
	WritingTile      AppendTile
	ReadCache        map[uint]*AppendTile
	MaxInCache       uint
	Backing          io.ReadWriteSeeker
}

// Creates a new append dataset. It initializes the internal data structures and sets up the backing store.
func NewAppendDataset(d Pixi, backing io.ReadWriteSeeker, maxInCache uint) (*AppendDataset, error) {
	appendSet := &AppendDataset{Pixi: d}
	appendSet.Backing = backing
	appendSet.ReadCache = make(map[uint]*AppendTile)
	appendSet.MaxInCache = maxInCache
	appendSet.WritingTileIndex = 0
	appendSet.WritingTile = AppendTile{Data: make([]byte, appendSet.TileSize(0))}
	appendSet.ReadCache[appendSet.WritingTileIndex] = &appendSet.WritingTile

	diskTileCount := appendSet.Tiles()
	if appendSet.Separated {
		diskTileCount *= len(appendSet.Fields)
	}
	appendSet.TileBytes = make([]int64, diskTileCount)
	appendSet.TileOffsets = make([]int64, diskTileCount)
	appendSet.TileOffsets[0] = appendSet.DiskDataStart()

	if err := WriteSummary(backing, appendSet.Pixi); err != nil {
		return nil, err
	}
	return appendSet, nil
}

func ReadAppend(r io.ReadWriteSeeker, ds Pixi, maxInCache uint) (AppendDataset, error) {
	appended := AppendDataset{Pixi: ds, ReadCache: make(map[uint]*AppendTile), Backing: r, MaxInCache: maxInCache}
	return appended, nil
}

func (d *AppendDataset) GetRawSample(dimIndices []uint) ([]byte, error) {
	if len(d.Dimensions) != len(dimIndices) {
		return nil, DimensionsError{len(d.Dimensions), len(dimIndices)}
	}

	tileIndex, inTileIndex := d.dimIndicesToTileIndices(dimIndices)

	if d.Separated {
		sample := make([]byte, d.SampleSize())
		sampleOffset := 0
		for fieldId, field := range d.Fields {
			fieldTile := tileIndex + uint(d.Tiles())*uint(fieldId)
			fieldOffset := inTileIndex * uint(field.Size())

			cached, err := d.getTile(fieldTile)
			if err != nil {
				return nil, err
			}

			copy(sample[sampleOffset:], cached.Data[fieldOffset:])
			sampleOffset += int(field.Size())
		}
		return sample, nil
	} else {
		fieldOffset := inTileIndex * uint(d.SampleSize())

		cached, err := d.getTile(tileIndex)
		if err != nil {
			return nil, err
		}

		return cached.Data[fieldOffset : fieldOffset+uint(d.SampleSize())], nil
	}
}

func (d *AppendDataset) GetSample(dimIndices []uint) ([]any, error) {
	raw, err := d.GetRawSample(dimIndices)
	if err != nil {
		return nil, err
	}

	sample := make([]any, len(d.Fields))
	fieldOffset := 0
	for fieldId, field := range d.Fields {
		fieldVal := field.Read(raw[fieldOffset:])
		sample[fieldId] = fieldVal

		fieldOffset += int(field.Size())
	}
	return sample, nil
}

func (d *AppendDataset) GetSampleField(dimIndices []uint, fieldId uint) (any, error) {
	if len(d.Dimensions) != len(dimIndices) {
		return nil, DimensionsError{len(d.Dimensions), len(dimIndices)}
	}

	tileIndex, inTileIndex := d.dimIndicesToTileIndices(dimIndices)
	fieldOffset := inTileIndex
	if d.Separated {
		tileIndex += uint(d.Tiles()) * uint(fieldId)
		fieldOffset *= uint(d.Fields[fieldId].Size())
	} else {
		fieldOffset *= uint(d.SampleSize())
		for _, field := range d.Fields[:fieldId] {
			fieldOffset += uint(field.Size())
		}
	}

	cached, err := d.getTile(tileIndex)
	if err != nil {
		return nil, err
	}

	return d.Fields[fieldId].Read(cached.Data[fieldOffset:]), nil
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
			err := d.writeTile(d.WritingTile.Data, d.WritingTileIndex)
			if err != nil {
				return err
			}
			d.WritingTileIndex += 1
			d.WritingTile = AppendTile{Data: make([]byte, d.TileSize(int(d.WritingTileIndex)))}
			err = d.addTileToCache(d.WritingTileIndex, d.WritingTile.Data)
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
	fieldOffset := inTileIndex
	if d.Separated {
		tileIndex += uint(d.Tiles()) * uint(fieldId)
		fieldOffset *= uint(d.Fields[fieldId].Size())
	} else {
		fieldOffset *= uint(d.SampleSize())
		for _, field := range d.Fields[:fieldId] {
			fieldOffset += uint(field.Size())
		}
	}

	// check if we need to move to the next tile or if we're out of range
	if tileIndex != d.WritingTileIndex {
		if tileIndex == d.WritingTileIndex+1 {
			err := d.writeTile(d.WritingTile.Data, d.WritingTileIndex)
			if err != nil {
				return err
			}
			d.WritingTileIndex += 1
			d.WritingTile = AppendTile{Data: make([]byte, d.TileSize(int(d.WritingTileIndex)))}
			err = d.addTileToCache(d.WritingTileIndex, d.WritingTile.Data)
			if err != nil {
				return err
			}
		} else {
			return RangeError{Specified: int(tileIndex), ValidMin: int(d.WritingTileIndex), ValidMax: int(d.WritingTileIndex)}
		}
	}

	d.Fields[fieldId].Write(d.WritingTile.Data[fieldOffset:], fieldVal)
	return nil
}

func (d *AppendDataset) Finalize() error {
	// last tile won't have been written yet
	err := d.writeTile(d.WritingTile.Data, d.WritingTileIndex)
	if err != nil {
		return err
	}
	d.WritingTileIndex += 1
	d.WritingTile = AppendTile{}

	_, err = d.Backing.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	err = WriteSummary(d.Backing, d.Pixi)
	if err != nil {
		return err
	}

	return d.addTileToCache(d.WritingTileIndex, nil)
}

func (d *AppendDataset) addTileToCache(tileIndex uint, data []byte) error {
	if len(d.ReadCache) >= int(d.MaxInCache) {
		err := d.evict()
		if err != nil {
			return err
		}
	}

	d.ReadCache[tileIndex] = &AppendTile{Data: data, Dirty: false}
	return nil
}

func (d *AppendDataset) getTile(tileIndex uint) (*AppendTile, error) {
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
	return cached, nil
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
	err = d.addTileToCache(tileIndex, read)
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
	d.Backing.Seek(d.TileOffsets[tileIndex], io.SeekStart)

	uncompressedLen := d.TileSize(int(tileIndex))

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

// This function takes in a byte slice and a tile index as input, and writes the contents of the slice to the backing storage at the specified tile offset.
// The function is responsible for handling both uncompressed and compressed data.
// If there was an issue with writing the tile, this function will return an error. Otherwise, it returns nil.
func (d *AppendDataset) writeTile(data []byte, tileIndex uint) error {
	offset := d.TileOffsets[tileIndex]
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
	if tileIndex < uint(d.DiskTiles())-1 {
		d.TileOffsets[tileIndex+1] = offset + int64(tileSize)
	}
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
*/
