package pixi

import (
	"compress/flate"
	"io"
)

type InMemoryDataset struct {
	DiskLayer
	TileSet [][]byte
}

func NewInMemoryDataset(l DiskLayer) (*InMemoryDataset, error) {
	memSet := &InMemoryDataset{DiskLayer: l}
	if l.Separated {
		memSet.TileSet = make([][]byte, memSet.Tiles()*len(l.Fields))
	} else {
		memSet.TileSet = make([][]byte, memSet.Tiles())
	}
	for tileInd := 0; tileInd < len(memSet.TileSet); tileInd++ {
		memSet.TileSet[tileInd] = make([]byte, memSet.DiskTileSize(tileInd))
	}
	return memSet, nil
}

func ReadInMemory(r io.ReadSeeker, ds DiskLayer) (InMemoryDataset, error) {
	inMem := InMemoryDataset{DiskLayer: ds}

	tiles := make([][]byte, len(ds.TileBytes))
	for tileInd := range ds.TileBytes {
		uncompressedLen := ds.DiskTileSize(tileInd)
		buf := make([]byte, uncompressedLen)
		_, err := r.Seek(ds.TileOffsets[tileInd], io.SeekStart)
		if err != nil {
			return inMem, err
		}

		switch ds.Compression {
		case CompressionNone:
			_, err := r.Read(buf)
			if err != nil {
				return inMem, err
			}
			tiles[tileInd] = buf
		case CompressionFlate:
			flateRdr := flate.NewReader(r)
			defer flateRdr.Close()
			_, err := flateRdr.Read(buf)
			if err != nil {
				return inMem, err
			}
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

	tileIndex, inTileIndex := d.dimIndicesToTileIndices(dimIndices)

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

	return d.Fields[fieldId].Read(d.TileSet[tileIndex][inTileIndex:]), nil
}

func (d *InMemoryDataset) SetSample(dimIndices []uint, sample []any) error {
	if len(d.Dimensions) != len(dimIndices) {
		return DimensionsError{len(d.Dimensions), len(dimIndices)}
	}

	tileIndex, inTileIndex := d.dimIndicesToTileIndices(dimIndices)

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

	d.Fields[fieldId].Write(d.TileSet[tileIndex][inTileIndex:], fieldVal)
	return nil
}

func (d *InMemoryDataset) dimIndicesToTileIndices(dimIndices []uint) (tileIndex uint, inTileIndex uint) {
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
