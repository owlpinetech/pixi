package pixi

import (
	"io"
	"sync"
)

type StoredLayer struct {
	lock   sync.RWMutex
	header *PixiHeader
	layer  *Layer
}

func NewStoredLayer(header *PixiHeader, layer *Layer) *StoredLayer {
	return &StoredLayer{
		header: header,
		layer:  layer,
	}
}

func (s *StoredLayer) Layer() *Layer {
	return s.layer
}

func (s *StoredLayer) Header() *PixiHeader {
	return s.header
}

func (s *StoredLayer) SampleAt(backing io.ReadSeeker, coord SampleCoordinate) ([]any, error) {
	tileSelector := coord.ToTileSelector(s.layer.Dimensions)
	sample := make([]any, len(s.layer.Fields))

	s.lock.Lock()
	defer s.lock.Unlock()
	if s.layer.Compression != CompressionNone {
		// inefficiently, we have to load the whole tile of the sample in order to extract it from compression
		if s.layer.Separated {
			for fieldIndex, field := range s.layer.Fields {
				fieldTile := tileSelector.Tile + s.layer.Dimensions.Tiles()*fieldIndex
				fieldOffset := tileSelector.InTile * field.Size()

				tileData, err := s.loadTile(backing, fieldTile)
				if err != nil {
					return nil, err
				}

				sample[fieldIndex] = field.BytesToValue(tileData[fieldOffset:], s.header.ByteOrder)
			}
		} else {
			fieldOffset := tileSelector.InTile * s.layer.Fields.Size()

			tileData, err := s.loadTile(backing, tileSelector.Tile)
			if err != nil {
				return nil, err
			}
			for i, field := range s.layer.Fields {
				sample[i] = field.BytesToValue(tileData[fieldOffset:], s.header.ByteOrder)
				fieldOffset += field.Size()
			}
		}
	} else {
		// with no compression, still inefficiently (but maybe less so), we can seek directly to the sample fields and read them only
		if s.layer.Separated {
			for fieldIndex, field := range s.layer.Fields {
				separatedTileIndex := tileSelector.Tile + s.layer.Dimensions.Tiles()*fieldIndex
				fieldFileOffset := s.layer.TileOffsets[separatedTileIndex] + int64(tileSelector.InTile*field.Size())

				fieldRead := make([]byte, field.Size())
				_, err := backing.Seek(fieldFileOffset, io.SeekStart)
				if err != nil {
					return nil, err
				}
				_, err = io.ReadFull(backing, fieldRead)
				if err != nil {
					return nil, err
				}
				sample[fieldIndex] = field.BytesToValue(fieldRead, s.header.ByteOrder)
			}
		} else {
			fieldTileOffset := tileSelector.InTile * s.layer.Fields.Size()
			fieldFileOffset := s.layer.TileOffsets[tileSelector.Tile] + int64(fieldTileOffset)
			sampleRead := make([]byte, s.layer.Fields.Size())
			_, err := backing.Seek(fieldFileOffset, io.SeekStart)
			if err != nil {
				return nil, err
			}
			_, err = io.ReadFull(backing, sampleRead)
			if err != nil {
				return nil, err
			}
			fieldOffset := 0
			for i, field := range s.layer.Fields {
				sample[i] = field.BytesToValue(sampleRead[fieldOffset:], s.header.ByteOrder)
				fieldOffset += field.Size()
			}
		}
	}

	return sample, nil
}

func (s *StoredLayer) FieldAt(backing io.ReadSeeker, coord SampleCoordinate, fieldIndex int) (any, error) {
	tileSelector := coord.ToTileSelector(s.layer.Dimensions)
	field := s.layer.Fields[fieldIndex]

	s.lock.Lock()
	defer s.lock.Unlock()
	if s.layer.Compression != CompressionNone {
		// inefficiently, we have to load the whole tile of the sample in order to extract it from compression
		if s.layer.Separated {
			fieldTile := tileSelector.Tile + s.layer.Dimensions.Tiles()*fieldIndex
			fieldOffset := tileSelector.InTile * field.Size()

			tileData, err := s.loadTile(backing, fieldTile)
			if err != nil {
				return nil, err
			}
			return field.BytesToValue(tileData[fieldOffset:], s.header.ByteOrder), nil
		} else {
			tileData, err := s.loadTile(backing, tileSelector.Tile)
			if err != nil {
				return nil, err
			}
			fieldOffset := tileSelector.InTile * s.layer.Fields.Size()
			for _, field := range s.layer.Fields[:fieldIndex] {
				fieldOffset += field.Size()
			}
			return field.BytesToValue(tileData[fieldOffset:], s.header.ByteOrder), nil
		}
	} else {
		// with no compression, still inefficiently (but maybe less so), we can seek directly to the sample fields and read them only
		if s.layer.Separated {
			separatedTileIndex := tileSelector.Tile + s.layer.Dimensions.Tiles()*fieldIndex
			fieldFileOffset := s.layer.TileOffsets[separatedTileIndex] + int64(tileSelector.InTile*field.Size())

			fieldRead := make([]byte, field.Size())
			_, err := backing.Seek(fieldFileOffset, io.SeekStart)
			if err != nil {
				return nil, err
			}
			_, err = io.ReadFull(backing, fieldRead)
			if err != nil {
				return nil, err
			}
			return field.BytesToValue(fieldRead, s.header.ByteOrder), nil
		} else {
			fieldTileOffset := tileSelector.InTile * s.layer.Fields.Size()
			for _, field := range s.layer.Fields[:fieldIndex] {
				fieldTileOffset += field.Size()
			}
			fieldFileOffset := s.layer.TileOffsets[tileSelector.Tile] + int64(fieldTileOffset)
			fieldRead := make([]byte, field.Size())
			_, err := backing.Seek(fieldFileOffset, io.SeekStart)
			if err != nil {
				return nil, err
			}
			_, err = io.ReadFull(backing, fieldRead)
			if err != nil {
				return nil, err
			}
			return field.BytesToValue(fieldRead, s.header.ByteOrder), nil
		}
	}
}

func (s *StoredLayer) SetSampleAt(backing io.WriteSeeker, coord SampleCoordinate, values []any) error {
	if s.layer.Compression != CompressionNone {
		panic("pixi: cannot set direct access sample on compressed layer")
	}
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

	s.lock.Lock()
	defer s.lock.Unlock()
	if s.layer.Separated {
		writeFieldOffset := 0
		for fieldIndex, field := range s.layer.Fields {
			separatedTileIndex := tileSelector.Tile + s.layer.Dimensions.Tiles()*fieldIndex
			fieldFileOffset := s.layer.TileOffsets[separatedTileIndex] + int64(tileSelector.InTile*field.Size())

			_, err := backing.Seek(fieldFileOffset, io.SeekStart)
			if err != nil {
				return err
			}
			_, err = backing.Write(raw[writeFieldOffset : writeFieldOffset+field.Size()])
			if err != nil {
				return err
			}
			writeFieldOffset += field.Size()
		}
	} else {
		fieldTileOffset := tileSelector.InTile * s.layer.Fields.Size()
		fieldFileOffset := s.layer.TileOffsets[tileSelector.Tile] + int64(fieldTileOffset)
		_, err := backing.Seek(fieldFileOffset, io.SeekStart)
		if err != nil {
			return err
		}
		_, err = backing.Write(raw)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *StoredLayer) SetFieldAt(backing io.WriteSeeker, coord SampleCoordinate, fieldIndex int, value any) error {
	if s.layer.Compression != CompressionNone {
		panic("cannot set field on compressed layer")
	}
	if fieldIndex < 0 || fieldIndex >= len(s.layer.Fields) {
		panic("pixi: field index out of range")
	}

	tileSelector := coord.ToTileSelector(s.layer.Dimensions)
	field := s.layer.Fields[fieldIndex]

	raw := make([]byte, field.Size())
	field.ValueToBytes(value, s.header.ByteOrder, raw)

	s.lock.Lock()
	defer s.lock.Unlock()
	if s.layer.Separated {
		separatedTileIndex := tileSelector.Tile + s.layer.Dimensions.Tiles()*fieldIndex
		fieldFileOffset := s.layer.TileOffsets[separatedTileIndex] + int64(tileSelector.InTile*field.Size())

		_, err := backing.Seek(fieldFileOffset, io.SeekStart)
		if err != nil {
			return err
		}
		_, err = backing.Write(raw)
		if err != nil {
			return err
		}
	} else {
		fieldTileOffset := tileSelector.InTile * s.layer.Fields.Size()
		for _, field := range s.layer.Fields[:fieldIndex] {
			fieldTileOffset += field.Size()
		}
		fieldFileOffset := s.layer.TileOffsets[tileSelector.Tile] + int64(fieldTileOffset)
		_, err := backing.Seek(fieldFileOffset, io.SeekStart)
		if err != nil {
			return err
		}
		_, err = backing.Write(raw)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *StoredLayer) loadTile(backing io.ReadSeeker, tileIndex int) ([]byte, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	chunk := make([]byte, c.layer.DiskTileSize(tileIndex))
	err := c.layer.ReadTile(backing, c.header, tileIndex, chunk)
	if err != nil {
		return nil, err
	}
	return chunk, nil
}
