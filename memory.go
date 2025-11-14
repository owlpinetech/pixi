package pixi

import (
	"io"
	"sync"
)

type MemoryLayer struct {
	lock    sync.RWMutex
	header  *PixiHeader
	layer   *Layer
	backing io.ReadWriteSeeker
	tiles   map[int][]byte
}

func NewMemoryLayer(backing io.ReadWriteSeeker, header *PixiHeader, layer *Layer) *MemoryLayer {
	return &MemoryLayer{
		header:  header,
		layer:   layer,
		backing: backing,
		tiles:   make(map[int][]byte),
	}
}

func (s *MemoryLayer) Layer() *Layer {
	return s.layer
}

func (s *MemoryLayer) Header() *PixiHeader {
	return s.header
}

func (s *MemoryLayer) SampleAt(coord SampleCoordinate) (Sample, error) {
	tileSelector := coord.ToTileSelector(s.layer.Dimensions)
	sample := make([]any, len(s.layer.Fields))

	if s.layer.Separated {
		for fieldIndex, field := range s.layer.Fields {
			fieldTile := tileSelector.Tile + s.layer.Dimensions.Tiles()*fieldIndex
			fieldOffset := tileSelector.InTile * field.Size()

			tileData, err := s.loadTile(fieldTile)
			if err != nil {
				return nil, err
			}

			sample[fieldIndex] = field.BytesToValue(tileData[fieldOffset:], s.header.ByteOrder)
		}
	} else {
		fieldOffset := tileSelector.InTile * s.layer.Fields.Size()

		tileData, err := s.loadTile(tileSelector.Tile)
		if err != nil {
			return nil, err
		}
		for i, field := range s.layer.Fields {
			sample[i] = field.BytesToValue(tileData[fieldOffset:], s.header.ByteOrder)
			fieldOffset += field.Size()
		}
	}

	return sample, nil
}

func (s *MemoryLayer) FieldAt(coord SampleCoordinate, fieldIndex int) (any, error) {
	tileSelector := coord.ToTileSelector(s.layer.Dimensions)
	field := s.layer.Fields[fieldIndex]

	if s.layer.Separated {
		fieldTile := tileSelector.Tile + s.layer.Dimensions.Tiles()*fieldIndex
		fieldOffset := tileSelector.InTile * field.Size()

		tileData, err := s.loadTile(fieldTile)
		if err != nil {
			return nil, err
		}
		return field.BytesToValue(tileData[fieldOffset:], s.header.ByteOrder), nil
	} else {
		tileData, err := s.loadTile(tileSelector.Tile)
		if err != nil {
			return nil, err
		}
		fieldOffset := tileSelector.InTile * s.layer.Fields.Size()
		for _, field := range s.layer.Fields[:fieldIndex] {
			fieldOffset += field.Size()
		}
		return field.BytesToValue(tileData[fieldOffset:], s.header.ByteOrder), nil
	}
}

func (s *MemoryLayer) SetSampleAt(coord SampleCoordinate, values Sample) error {
	if len(values) != len(s.layer.Fields) {
		panic("pixi: values length does not match field count")
	}

	tileSelector := coord.ToTileSelector(s.layer.Dimensions)
	raw := make([]byte, s.layer.Fields.Size())
	fieldOffset := 0
	for i, field := range s.layer.Fields {
		field.ValueToBytes(values[i], s.header.ByteOrder, raw[fieldOffset:])
		fieldOffset += field.Size()
	}

	if s.layer.Separated {
		for fieldIndex, field := range s.layer.Fields {
			fieldTile := tileSelector.Tile + s.layer.Dimensions.Tiles()*fieldIndex
			fieldOffset := tileSelector.InTile * field.Size()

			tileData, err := s.loadTile(fieldTile)
			if err != nil {
				return err
			}
			field.ValueToBytes(values[fieldIndex], s.header.ByteOrder, tileData[fieldOffset:])
		}
	} else {
		fieldOffset := tileSelector.InTile * s.layer.Fields.Size()

		tileData, err := s.loadTile(tileSelector.Tile)
		if err != nil {
			return err
		}
		for i, field := range s.layer.Fields {
			field.ValueToBytes(values[i], s.header.ByteOrder, tileData[fieldOffset:])
			fieldOffset += field.Size()
		}
	}

	return nil
}

func (s *MemoryLayer) SetFieldAt(coord SampleCoordinate, fieldIndex int, value any) error {
	if fieldIndex < 0 || fieldIndex >= len(s.layer.Fields) {
		panic("pixi: field index out of range")
	}

	tileSelector := coord.ToTileSelector(s.layer.Dimensions)
	field := s.layer.Fields[fieldIndex]

	raw := make([]byte, field.Size())
	field.ValueToBytes(value, s.header.ByteOrder, raw)

	if s.layer.Separated {
		fieldTile := tileSelector.Tile + s.layer.Dimensions.Tiles()*fieldIndex
		fieldOffset := tileSelector.InTile * field.Size()

		tileData, err := s.loadTile(fieldTile)
		if err != nil {
			return err
		}
		field.ValueToBytes(value, s.header.ByteOrder, tileData[fieldOffset:])
	} else {
		tileData, err := s.loadTile(tileSelector.Tile)
		if err != nil {
			return err
		}
		fieldOffset := tileSelector.InTile * s.layer.Fields.Size()
		for _, field := range s.layer.Fields[:fieldIndex] {
			fieldOffset += field.Size()
		}
		field.ValueToBytes(value, s.header.ByteOrder, tileData[fieldOffset:])
	}
	return nil
}

func (s *MemoryLayer) Flush() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	for tileIndex, tileData := range s.tiles {
		if s.layer.TileBytes[tileIndex] != 0 {
			if s.layer.Compression != CompressionNone {
				panic("pixi: cannot overwrite flush compressed layer")
			}
			err := s.layer.OverwriteTile(s.backing, s.header, tileIndex, tileData)
			if err != nil {
				return err
			}
		} else {
			// always write new tiles at the end of the file
			_, err := s.backing.Seek(0, io.SeekEnd)
			if err != nil {
				return err
			}
			err = s.layer.WriteTile(s.backing, s.header, tileIndex, tileData)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *MemoryLayer) loadTile(tileIndex int) ([]byte, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if chunk, exists := c.tiles[tileIndex]; exists {
		return chunk, nil
	}

	// if the tile has already been written
	chunk := make([]byte, c.layer.DiskTileSize(tileIndex))
	if c.layer.TileBytes[tileIndex] != 0 {
		err := c.layer.ReadTile(c.backing, c.header, tileIndex, chunk)
		if err != nil {
			return nil, err
		}
	}
	c.tiles[tileIndex] = chunk
	return chunk, nil
}
