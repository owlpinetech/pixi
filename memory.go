package pixi

import (
	"compress/gzip"
	"io"
)

type InMemoryDataset struct {
	DataSet
	TileSet [][]byte
}

func NewInMemoryDataset(separated bool, compression Compression, dims []Dimension, fields []Field) (*InMemoryDataset, error) {
	memSet := &InMemoryDataset{}
	memSet.Separated = separated
	memSet.Compression = compression
	memSet.Dimensions = dims
	memSet.Fields = fields
	if separated {
		memSet.TileSet = make([][]byte, memSet.Tiles()*len(fields))
	} else {
		memSet.TileSet = make([][]byte, memSet.Tiles())
	}
	for tileInd := 0; tileInd < len(memSet.TileSet); tileInd++ {
		memSet.TileSet[tileInd] = make([]byte, memSet.TileSize(tileInd))
	}
	return memSet, nil
}

func ReadInMemory(r io.ReadSeeker, ds DataSet) (InMemoryDataset, error) {
	inMem := InMemoryDataset{DataSet: ds}

	tiles := make([][]byte, len(ds.TileBytes))
	r.Seek(ds.Offset, io.SeekStart)
	for tileInd := range ds.TileBytes {
		uncompressedLen := ds.TileSize(tileInd)
		buf := make([]byte, uncompressedLen)

		switch ds.Compression {
		case CompressionNone:
			_, err := r.Read(buf)
			if err != nil {
				return inMem, err
			}
			tiles[tileInd] = buf
		case CompressionGzip:
			gzRdr, err := gzip.NewReader(r)
			if err != nil {
				return inMem, err
			}
			_, err = gzRdr.Read(buf)
			if err != nil {
				gzRdr.Close()
				return inMem, err
			}
			gzRdr.Close()
			tiles[tileInd] = buf
		}
	}
	inMem.TileSet = tiles
	return inMem, nil
}

func (d *InMemoryDataset) GetSample(dimIndices []uint) ([]any, error) {
	if len(d.Dimensions) != len(dimIndices) {
		return nil, DimensionsError{len(d.Dimensions), len(dimIndices)}
	}

	tileIndex := uint(0)
	inTileIndex := uint(0)
	mul := uint(1)
	for dInd, index := range dimIndices {
		tileIndex += (index / uint(d.Dimensions[dInd].TileSize)) * mul
		inTileIndex += (index % uint(d.Dimensions[dInd].TileSize))
		mul *= uint(d.Dimensions[dInd].TileSize)
	}

	sample := make([]any, len(d.Fields))

	if d.Separated {
		for fieldId, field := range d.Fields {
			fieldTile := tileIndex + uint(d.Tiles())*uint(fieldId)
			fieldOffset := inTileIndex * uint(field.Size())
			fieldVal := field.Read(d.TileSet[fieldTile][fieldOffset:])
			sample[fieldId] = fieldVal
		}
	} else {
		inTileIndex *= uint(d.SampleSize())
		data := d.TileSet[tileIndex]
		for fieldId, field := range d.Fields {
			fieldVal := field.Read(data[inTileIndex:])
			sample[fieldId] = fieldVal

			inTileIndex += uint(field.Size())
		}
	}

	return sample, nil
}

func (d *InMemoryDataset) GetSampleField(dimIndices []uint, fieldId uint) (any, error) {
	if len(d.Dimensions) != len(dimIndices) {
		return nil, DimensionsError{len(d.Dimensions), len(dimIndices)}
	}

	tileIndex := uint(0)
	inTileIndex := uint(0)
	mul := uint(1)
	for dInd, index := range dimIndices {
		tileIndex += (index / uint(d.Dimensions[dInd].TileSize)) * mul
		inTileIndex += (index % uint(d.Dimensions[dInd].TileSize))
		mul *= uint(d.Dimensions[dInd].TileSize)
	}
	if d.Separated {
		tileIndex += uint(d.Tiles()) * uint(fieldId)
		inTileIndex *= uint(d.Fields[fieldId].Size())
	} else {
		inTileIndex *= uint(d.SampleSize())
	}

	return d.Fields[fieldId].Read(d.TileSet[tileIndex][inTileIndex:]), nil
}

func (d *InMemoryDataset) SetSample(dimIndices []uint, sample []any) error {
	if len(d.Dimensions) != len(dimIndices) {
		return DimensionsError{len(d.Dimensions), len(dimIndices)}
	}

	tileIndex := uint(0)
	inTileIndex := uint(0)
	mul := uint(1)
	for dInd, index := range dimIndices {
		tileIndex += (index / uint(d.Dimensions[dInd].TileSize)) * mul
		inTileIndex += (index % uint(d.Dimensions[dInd].TileSize))
		mul *= uint(d.Dimensions[dInd].TileSize)
	}

	if d.Separated {
		for fieldId, field := range d.Fields {
			fieldTile := tileIndex + uint(d.Tiles())*uint(fieldId)
			fieldOffset := inTileIndex * uint(field.Size())
			field.Write(d.TileSet[fieldTile][fieldOffset:], sample[fieldId])
		}
	} else {
		inTileIndex *= uint(d.SampleSize())
		data := d.TileSet[tileIndex]
		for fieldId, field := range d.Fields {
			field.Write(data[inTileIndex:], sample[fieldId])
			inTileIndex += uint(field.Size())
		}
	}

	return nil
}

func (d *InMemoryDataset) SetSampleField(dimIndices []uint, fieldId uint, fieldVal any) error {
	if len(d.Dimensions) != len(dimIndices) {
		return DimensionsError{len(d.Dimensions), len(dimIndices)}
	}

	tileIndex := uint(0)
	inTileIndex := uint(0)
	mul := uint(1)
	for dInd, index := range dimIndices {
		tileIndex += (index / uint(d.Dimensions[dInd].TileSize)) * mul
		inTileIndex += (index % uint(d.Dimensions[dInd].TileSize))
		mul *= uint(d.Dimensions[dInd].TileSize)
	}
	if d.Separated {
		tileIndex += uint(d.Tiles()) * uint(fieldId)
		inTileIndex *= uint(d.Fields[fieldId].Size())
	} else {
		inTileIndex *= uint(d.SampleSize())
	}

	d.Fields[fieldId].Write(d.TileSet[tileIndex][inTileIndex:], fieldVal)
	return nil
}
