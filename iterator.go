package pixi

import (
	"io"
	"sync"

	"github.com/owlpinetech/pixi/internal/preload"
)

const (
	nonSeparatedKey = -1
)

type TileOrderReadIterator struct {
	backing io.ReadSeeker
	header  *Header
	layer   *Layer

	tile         int
	sampleInTile int
	preloader    *preload.Preloader[map[int][]byte]

	tiles        map[int][]byte
	currentError error
}

func NewTileOrderReadIterator(backing io.ReadSeeker, header *Header, layer *Layer) *TileOrderReadIterator {
	iterator := &TileOrderReadIterator{
		backing:      backing,
		header:       header,
		layer:        layer,
		sampleInTile: -1, // so first Next() goes to 0
		tiles:        make(map[int][]byte),
	}
	iterator.preloader = preload.NewPreloader(iterator.readTiles, 2)
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
		// check if we are done
		if t.tile >= t.layer.Dimensions.Tiles() {
			return false
		} else {
			// load the next tile (or tiles, if separated)
			if t.tile < t.layer.Dimensions.Tiles()-1 {
				t.preloader.Notify()
			}
			t.tiles, t.currentError = t.preloader.Next()
		}
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
		if t.layer.Fields[fieldIndex].Type == FieldBool {
			return UnpackBool(tileData, t.sampleInTile)
		} else {
			inTileOffset := t.sampleInTile * t.layer.Fields[fieldIndex].Size()
			return t.layer.Fields[fieldIndex].Value(tileData[inTileOffset:], t.header.ByteOrder)
		}
	} else {
		tileData := t.tiles[nonSeparatedKey]
		inTileOffset := t.sampleInTile * t.layer.Fields.Size()
		fieldOffset := t.layer.Fields.Offset(fieldIndex)
		return t.layer.Fields[fieldIndex].Value(tileData[inTileOffset+fieldOffset:], t.header.ByteOrder)
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
			if field.Type == FieldBool {
				sample[fieldIndex] = UnpackBool(tileData, t.sampleInTile)
			} else {
				inTileOffset := t.sampleInTile * field.Size()
				sample[fieldIndex] = field.Value(tileData[inTileOffset:], t.header.ByteOrder)
			}
		}
	} else {
		tileData := t.tiles[nonSeparatedKey]
		inTileOffset := t.sampleInTile * t.layer.Fields.Size()
		for fieldIndex, field := range t.layer.Fields {
			sample[fieldIndex] = field.Value(tileData[inTileOffset:], t.header.ByteOrder)
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
			tileData := make([]byte, t.layer.DiskTileSize(fieldTile))
			err := t.layer.ReadTile(t.backing, t.header, fieldTile, tileData)
			if err != nil {
				return nil, err
			}
			result[fieldIndex] = tileData
		}
	} else {
		tileData := make([]byte, t.layer.DiskTileSize(tileIndex))
		err := t.layer.ReadTile(t.backing, t.header, tileIndex, tileData)
		if err != nil {
			return nil, err
		}
		result[nonSeparatedKey] = tileData
	}
	return result, nil
}

type TileOrderWriteIterator struct {
	backing io.WriteSeeker
	header  *Header
	layer   *Layer

	tile         int
	sampleInTile int

	wg           sync.WaitGroup
	writeLock    sync.RWMutex
	writeQueue   chan map[int][]byte
	currentError error

	tiles map[int][]byte
}

func NewTileOrderWriteIterator(backing io.WriteSeeker, header *Header, layer *Layer) *TileOrderWriteIterator {
	iterator := &TileOrderWriteIterator{
		backing: backing,
		header:  header,
		layer:   layer,

		sampleInTile: -1, // so first Next() goes to 0

		writeQueue: make(chan map[int][]byte, 100),

		tiles: make(map[int][]byte),
	}

	if layer.Separated {
		for fieldIndex := range layer.Fields {
			tileSize := layer.DiskTileSize(layer.Dimensions.Tiles() * fieldIndex)
			iterator.tiles[fieldIndex] = make([]byte, tileSize)
		}
	} else {
		tileSize := layer.DiskTileSize(0)
		iterator.tiles[nonSeparatedKey] = make([]byte, tileSize)
	}

	iterator.wg.Go(func() {
		tileIndex := 0
		for tiles := range iterator.writeQueue {
			err := iterator.writeTiles(tiles, tileIndex)
			if err != nil {
				iterator.writeLock.Lock()
				iterator.currentError = err
				iterator.writeLock.Unlock()
				return
			}
			tileIndex += 1
		}
	})

	return iterator
}

func (t *TileOrderWriteIterator) Layer() *Layer {
	return t.layer
}

func (t *TileOrderWriteIterator) Done() {
	close(t.writeQueue)
	t.wg.Wait()
}

func (t *TileOrderWriteIterator) Error() error {
	t.writeLock.RLock()
	defer t.writeLock.RUnlock()
	return t.currentError
}

func (t *TileOrderWriteIterator) Next() bool {
	if t.Error() != nil {
		return false
	}

	// advance to next sample
	t.sampleInTile += 1
	if t.sampleInTile >= t.layer.Dimensions.TileSamples() {
		t.sampleInTile = 0
		t.tile += 1

		t.writeQueue <- t.tiles
		t.tiles = make(map[int][]byte)

		// check if we are done
		if t.tile >= t.layer.Dimensions.Tiles() {
			return false
		} else {
			// load the next tile (or tiles, if separated)
			if t.layer.Separated {
				for fieldIndex := range t.layer.Fields {
					tileSize := t.layer.DiskTileSize(t.tile + t.layer.Dimensions.Tiles()*fieldIndex)
					t.tiles[fieldIndex] = make([]byte, tileSize)
				}
			} else {
				tileSize := t.layer.DiskTileSize(t.tile)
				t.tiles[nonSeparatedKey] = make([]byte, tileSize)
			}
		}
	}

	return true
}

func (t *TileOrderWriteIterator) Coordinate() SampleCoordinate {
	tileSelector := TileSelector{
		Tile:   t.tile,
		InTile: t.sampleInTile,
	}
	// TODO: track and increment coordinates directly instead of converting from tile selector each time
	return tileSelector.
		ToTileCoordinate(t.layer.Dimensions).
		ToSampleCoordinate(t.layer.Dimensions)
}

func (t *TileOrderWriteIterator) SetField(fieldIndex int, value any) {
	if t.Error() != nil {
		return
	}

	// Update Min/Max for the field
	t.layer.Fields[fieldIndex].UpdateMinMax(value)

	if t.layer.Separated {
		tileData := t.tiles[fieldIndex]
		if t.layer.Fields[fieldIndex].Type == FieldBool {
			PackBool(value.(bool), tileData, t.sampleInTile)
		} else {
			inTileOffset := t.sampleInTile * t.layer.Fields[fieldIndex].Size()
			t.layer.Fields[fieldIndex].PutValue(value, t.header.ByteOrder, tileData[inTileOffset:])
		}
	} else {
		tileData := t.tiles[nonSeparatedKey]
		inTileOffset := t.sampleInTile * t.layer.Fields.Size()
		fieldOffset := t.layer.Fields.Offset(fieldIndex)
		t.layer.Fields[fieldIndex].PutValue(value, t.header.ByteOrder, tileData[inTileOffset+fieldOffset:])
	}
}

func (t *TileOrderWriteIterator) SetSample(value Sample) {
	if t.Error() != nil {
		return
	}

	// Update Min/Max for all fields in the sample
	for fieldIndex, fieldValue := range value {
		t.layer.Fields[fieldIndex].UpdateMinMax(fieldValue)
	}

	if t.layer.Separated {
		for fieldIndex, field := range t.layer.Fields {
			tileData := t.tiles[fieldIndex]
			if field.Type == FieldBool {
				PackBool(value[fieldIndex].(bool), tileData, t.sampleInTile)
			} else {
				inTileOffset := t.sampleInTile * field.Size()
				field.PutValue(value[fieldIndex], t.header.ByteOrder, tileData[inTileOffset:])
			}
		}
	} else {
		tileData := t.tiles[nonSeparatedKey]
		inTileOffset := t.sampleInTile * t.layer.Fields.Size()
		for fieldIndex, field := range t.layer.Fields {
			field.PutValue(value[fieldIndex], t.header.ByteOrder, tileData[inTileOffset:])
			inTileOffset += field.Size()
		}
	}
}

func (t *TileOrderWriteIterator) writeTiles(tiles map[int][]byte, tileIndex int) error {
	if t.layer.Separated {
		for fieldIndex := range t.layer.Fields {
			fieldTile := tileIndex + t.layer.Dimensions.Tiles()*fieldIndex
			err := t.layer.WriteTile(t.backing, t.header, fieldTile, tiles[fieldIndex])
			if err != nil {
				return err
			}
		}
		return nil
	} else {
		return t.layer.WriteTile(t.backing, t.header, tileIndex, tiles[nonSeparatedKey])
	}
}
