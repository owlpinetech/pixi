package pixi

import (
	"io"
	"sync"
	"time"
)

type CachedLayerReadCache interface {
	LayerExtension
	Get(tile int) ([]byte, error)
}

type CachedLayerCache interface {
	CachedLayerReadCache
	SetFragment(tile int, tileOffset int, data []byte) error
	Flush() error
}

type LayerFifoCacheTile struct {
	age  time.Time
	data []byte
}

type LayerReadFifoCache struct {
	cacheLock sync.RWMutex
	backing   io.ReadSeeker
	layer     *Layer
	cache     map[int]LayerFifoCacheTile
	maxSize   int
}

func NewLayerReadFifoCache(backing io.ReadSeeker, layer *Layer, maxSize int) *LayerReadFifoCache {
	return &LayerReadFifoCache{
		backing: backing,
		layer:   layer,
		cache:   make(map[int]LayerFifoCacheTile),
		maxSize: maxSize,
	}
}

func (c *LayerReadFifoCache) Layer() *Layer {
	return c.layer
}

func (c *LayerReadFifoCache) Get(tile int) ([]byte, error) {
	c.cacheLock.RLock()
	cached, found := c.cache[tile]
	tileOffset := c.layer.TileOffsets[tile]
	tileSize := c.layer.TileBytes[tile]
	c.cacheLock.RUnlock()
	if found {
		return cached.data, nil
	}

	data := make([]byte, tileSize)
	c.cacheLock.Lock()
	_, err := c.backing.Seek(tileOffset, io.SeekStart)
	if err != nil {
		return nil, err
	}
	_, err = c.backing.Read(data)
	if err != nil {
		return nil, err
	}

	if len(c.cache) >= c.maxSize {
		var oldestTile int
		var oldestTime time.Time
		for t, entry := range c.cache {
			if oldestTime.IsZero() || entry.age.Before(oldestTime) {
				oldestTime = entry.age
				oldestTile = t
			}
		}
		delete(c.cache, oldestTile)
	}
	c.cache[tile] = LayerFifoCacheTile{
		age:  time.Now(),
		data: data,
	}
	c.cacheLock.Unlock()

	return data, nil
}

type LayerFifoCache struct {
	LayerReadFifoCache
	backing io.ReadWriteSeeker
}

func NewLayerFifoCache(backing io.ReadWriteSeeker, layer *Layer, maxSize int) *LayerFifoCache {
	return &LayerFifoCache{
		LayerReadFifoCache: LayerReadFifoCache{
			backing: backing,
			layer:   layer,
			cache:   make(map[int]LayerFifoCacheTile),
			maxSize: maxSize,
		},
		backing: backing,
	}
}

func (c *LayerFifoCache) SetFragment(tile int, tileOffset int, data []byte) error {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	cached, found := c.cache[tile]
	if found {
		copy(cached.data[tileOffset:], data)
		c.cache[tile] = LayerFifoCacheTile{
			age:  time.Now(),
			data: cached.data,
		}
	} else {
		tileData := make([]byte, c.layer.TileBytes[tile])
		_, err := c.backing.Seek(c.layer.TileOffsets[tile], io.SeekStart)
		if err != nil {
			return err
		}
		_, err = c.backing.Read(tileData)
		if err != nil {
			return err
		}
		copy(tileData[tileOffset:], data)

		if len(c.cache) >= c.maxSize {
			var oldestTile int
			var oldestTime time.Time
			for t, entry := range c.cache {
				if oldestTime.IsZero() || entry.age.Before(oldestTime) {
					oldestTime = entry.age
					oldestTile = t
				}
			}

			oldest := c.cache[oldestTile]
			_, err := c.backing.Seek(c.layer.TileOffsets[oldestTile], io.SeekStart)
			if err != nil {
				return err
			}
			_, err = c.backing.Write(oldest.data)
			if err != nil {
				return err
			}
			delete(c.cache, oldestTile)
		}
		c.cache[tile] = LayerFifoCacheTile{
			age:  time.Now(),
			data: tileData,
		}
	}

	return nil
}

func (c *LayerFifoCache) Flush() error {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	for tile, entry := range c.cache {
		_, err := c.backing.Seek(c.layer.TileOffsets[tile], io.SeekStart)
		if err != nil {
			return err
		}
		_, err = c.backing.Write(entry.data)
		if err != nil {
			return err
		}
	}
	return nil
}

type ReadCachedLayer struct {
	header *PixiHeader
	cache  CachedLayerReadCache
}

type WriteCachedLayer struct {
	header *PixiHeader
	cache  CachedLayerCache
}

type CachedLayer struct {
	ReadCachedLayer
	WriteCachedLayer
}

func NewReadCachedLayer(header *PixiHeader, cache CachedLayerReadCache) *ReadCachedLayer {
	return &ReadCachedLayer{
		header: header,
		cache:  cache,
	}
}

func NewCachedLayer(header *PixiHeader, cache CachedLayerCache) *CachedLayer {
	return &CachedLayer{
		ReadCachedLayer:  ReadCachedLayer{header: header, cache: cache},
		WriteCachedLayer: WriteCachedLayer{header: header, cache: cache},
	}
}

func (s *ReadCachedLayer) Layer() *Layer {
	return s.cache.Layer()
}

func (s *ReadCachedLayer) Header() *PixiHeader {
	return s.header
}

func (s *WriteCachedLayer) Layer() *Layer {
	return s.cache.Layer()
}

func (s *WriteCachedLayer) Header() *PixiHeader {
	return s.header
}

func (s *CachedLayer) Layer() *Layer {
	return s.ReadCachedLayer.cache.Layer()
}

func (s *CachedLayer) Header() *PixiHeader {
	return s.ReadCachedLayer.header
}

func (s *ReadCachedLayer) SampleAt(coord SampleCoordinate) ([]any, error) {
	tileSelector := coord.ToTileSelector(s.Layer().Dimensions)
	sample := make([]any, len(s.Layer().Fields))

	if s.Layer().Separated {
		for fieldIndex, field := range s.Layer().Fields {
			fieldTile := tileSelector.Tile + s.Layer().Dimensions.Tiles()*fieldIndex
			fieldOffset := tileSelector.InTile * field.Size()

			tileData, err := s.cache.Get(fieldTile)
			if err != nil {
				return nil, err
			}

			sample[fieldIndex] = field.BytesToValue(tileData[fieldOffset:], s.header.ByteOrder)
		}
	} else {
		fieldOffset := tileSelector.InTile * s.Layer().Fields.Size()

		tileData, err := s.cache.Get(tileSelector.Tile)
		if err != nil {
			return nil, err
		}
		for i, field := range s.Layer().Fields {
			sample[i] = field.BytesToValue(tileData[fieldOffset:], s.header.ByteOrder)
			fieldOffset += field.Size()
		}
	}

	return sample, nil
}

func (s *ReadCachedLayer) FieldAt(coord SampleCoordinate, fieldIndex int) (any, error) {
	tileSelector := coord.ToTileSelector(s.Layer().Dimensions)
	field := s.Layer().Fields[fieldIndex]

	if s.Layer().Separated {
		fieldTile := tileSelector.Tile + s.Layer().Dimensions.Tiles()*fieldIndex
		fieldOffset := tileSelector.InTile * field.Size()

		tileData, err := s.cache.Get(fieldTile)
		if err != nil {
			return nil, err
		}
		return field.BytesToValue(tileData[fieldOffset:], s.header.ByteOrder), nil
	} else {
		tileData, err := s.cache.Get(tileSelector.Tile)
		if err != nil {
			return nil, err
		}
		fieldOffset := tileSelector.InTile * s.Layer().Fields.Size()
		for _, field := range s.Layer().Fields[:fieldIndex] {
			fieldOffset += field.Size()
		}
		return field.BytesToValue(tileData[fieldOffset:], s.header.ByteOrder), nil
	}
}

func (s *WriteCachedLayer) SetSampleAt(coord SampleCoordinate, values []any) error {
	if s.Layer().Compression != CompressionNone {
		panic("pixi: cannot set direct access sample on compressed layer")
	}
	if len(values) != len(s.Layer().Fields) {
		panic("pixi: values length does not match field count")
	}

	tileSelector := coord.ToTileSelector(s.Layer().Dimensions)
	raw := make([]byte, s.Layer().Fields.Size())
	fieldOffset := 0
	for i, field := range s.Layer().Fields {
		field.ValueToBytes(values[i], s.header.ByteOrder, raw[fieldOffset:])
		fieldOffset += field.Size()
	}

	if s.Layer().Separated {
		writeFieldOffset := 0
		for fieldIndex, field := range s.Layer().Fields {
			separatedTileIndex := tileSelector.Tile + s.Layer().Dimensions.Tiles()*fieldIndex
			fieldInTileOffset := tileSelector.InTile * field.Size()

			err := s.cache.SetFragment(separatedTileIndex, fieldInTileOffset, raw[writeFieldOffset:writeFieldOffset+field.Size()])
			if err != nil {
				return err
			}
			writeFieldOffset += field.Size()
		}
	} else {
		fieldInTileOffset := tileSelector.InTile * s.Layer().Fields.Size()
		err := s.cache.SetFragment(tileSelector.Tile, fieldInTileOffset, raw)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *WriteCachedLayer) SetFieldAt(coord SampleCoordinate, fieldIndex int, value any) error {
	if s.Layer().Compression != CompressionNone {
		panic("cannot set field on compressed layer")
	}
	if fieldIndex < 0 || fieldIndex >= len(s.Layer().Fields) {
		panic("pixi: field index out of range")
	}

	tileSelector := coord.ToTileSelector(s.Layer().Dimensions)
	field := s.Layer().Fields[fieldIndex]

	raw := make([]byte, field.Size())
	field.ValueToBytes(value, s.header.ByteOrder, raw)

	if s.Layer().Separated {
		separatedTileIndex := tileSelector.Tile + s.Layer().Dimensions.Tiles()*fieldIndex
		fieldInTileOffset := tileSelector.InTile * field.Size()
		err := s.cache.SetFragment(separatedTileIndex, fieldInTileOffset, raw)
		if err != nil {
			return err
		}
	} else {
		fieldTileOffset := tileSelector.InTile * s.Layer().Fields.Size()
		for _, field := range s.Layer().Fields[:fieldIndex] {
			fieldTileOffset += field.Size()
		}
		s.cache.SetFragment(tileSelector.Tile, fieldTileOffset, raw)
	}

	return nil
}
