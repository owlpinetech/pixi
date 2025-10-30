package pixi

import (
	"io"
)

const (
	nonSeparatedKey = -1
)

type TileOrderReadIterator struct {
	backing io.ReadSeeker
	header  *PixiHeader
	layer   *Layer

	tile         int
	sampleInTile int
	preloader    *Preloader[map[int][]byte]

	tiles        map[int][]byte
	currentError error
}

func NewTileOrderSampleReadIterator(backing io.ReadSeeker, header *PixiHeader, layer *Layer) *TileOrderReadIterator {
	iterator := &TileOrderReadIterator{
		backing:      backing,
		header:       header,
		layer:        layer,
		sampleInTile: -1, // so first Next() goes to 0
	}
	iterator.preloader = NewPreloader(iterator.readTiles, 2)
	// notify twice so we load more than we need right away
	iterator.preloader.Notify()
	iterator.preloader.Notify()
	iterator.preloader.Start()

	iterator.tiles, iterator.currentError = iterator.preloader.Next()

	return iterator
}

func (t *TileOrderReadIterator) Layer() *Layer {
	return t.layer
}

func (t *TileOrderReadIterator) Done() {
	t.preloader.Stop()
}

func (t *TileOrderReadIterator) Error() error {
	return t.currentError
}

func (t *TileOrderReadIterator) Next() bool {
	if t.currentError != nil {
		return false
	}

	// advance to next sample
	t.sampleInTile += 1
	if t.sampleInTile >= t.layer.Dimensions.TileSamples() {
		t.sampleInTile = 0
		t.tile += 1
		// load the next tile (or tiles, if separated)
		if t.tile < t.layer.Dimensions.Tiles() {
			t.preloader.Notify()
		}
		t.tiles, t.currentError = t.preloader.Next()
	}

	// check if we are done
	if t.tile >= t.layer.Dimensions.Tiles() {
		return false
	}

	return true
}

func (t *TileOrderReadIterator) Coordinate() SampleCoordinate {
	tileSelector := TileSelector{
		Tile:   t.tile,
		InTile: t.sampleInTile,
	}
	// TODO: track and increment coordinates directly instead of converting from tile selector each time
	return tileSelector.
		ToTileCoordinate(t.layer.Dimensions).
		ToSampleCoordinate(t.layer.Dimensions)
}

func (t *TileOrderReadIterator) Field(fieldIndex int) any {
	if t.currentError != nil {
		return nil
	}

	if t.layer.Separated {
		tileData := t.tiles[fieldIndex]
		inTileOffset := t.sampleInTile * t.layer.Fields[fieldIndex].Size()
		return t.layer.Fields[fieldIndex].BytesToValue(tileData[inTileOffset:], t.header.ByteOrder)
	} else {
		tileData := t.tiles[nonSeparatedKey]
		inTileOffset := t.sampleInTile * t.layer.Fields.Size()
		fieldOffset := t.layer.Fields.Offset(fieldIndex)
		return t.layer.Fields[fieldIndex].BytesToValue(tileData[inTileOffset+fieldOffset:], t.header.ByteOrder)
	}
}

func (t *TileOrderReadIterator) Sample() Sample {
	if t.currentError != nil {
		return nil
	}

	sample := make([]any, len(t.layer.Fields))
	if t.layer.Separated {
		for fieldIndex, field := range t.layer.Fields {
			tileData := t.tiles[fieldIndex]
			inTileOffset := t.sampleInTile * field.Size()
			sample[fieldIndex] = field.BytesToValue(tileData[inTileOffset:], t.header.ByteOrder)
		}
	} else {
		tileData := t.tiles[nonSeparatedKey]
		inTileOffset := t.sampleInTile * t.layer.Fields.Size()
		for fieldIndex, field := range t.layer.Fields {
			sample[fieldIndex] = field.BytesToValue(tileData[inTileOffset:], t.header.ByteOrder)
			inTileOffset += field.Size()
		}
	}

	return sample
}

func (t *TileOrderReadIterator) readTiles(tileIndex int) (map[int][]byte, error) {
	result := make(map[int][]byte)
	if t.layer.Separated {
		for fieldIndex := range t.layer.Fields {
			fieldTile := tileIndex + t.layer.Dimensions.Tiles()*fieldIndex
			tileData := make([]byte, t.layer.TileBytes[fieldTile])
			_, err := t.backing.Seek(t.layer.TileOffsets[fieldTile], io.SeekStart)
			if err != nil {
				return nil, err
			}
			_, err = io.ReadFull(t.backing, tileData)
			if err != nil {
				return nil, err
			}
			result[fieldIndex] = tileData
		}
	} else {
		tileData := make([]byte, t.layer.TileBytes[tileIndex])
		_, err := t.backing.Seek(t.layer.TileOffsets[tileIndex], io.SeekStart)
		if err != nil {
			return nil, err
		}
		_, err = io.ReadFull(t.backing, tileData)
		if err != nil {
			return nil, err
		}
		result[nonSeparatedKey] = tileData
	}
	return result, nil
}
