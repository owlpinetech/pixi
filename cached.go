package pixi

import (
	"io"
	"sync"
	"time"
)

type CachedLayerReadCache interface {
	LayerExtension
	Header() *PixiHeader
	Get(tile int) ([]byte, error)
}

type CachedLayerCache interface {
	CachedLayerReadCache
	SetFragment(tile int, tileOffset int, data []byte) error
	SetBit(tile int, bitIndex int, value bool) error
	Flush() error
}

type LayerFifoCacheTile struct {
	age  time.Time
	data []byte
}

type LayerReadFifoCache struct {
	cacheLock sync.RWMutex
	backing   io.ReadSeeker
	header    *PixiHeader
	layer     *Layer
	cache     map[int]LayerFifoCacheTile
	maxSize   int
}

func NewLayerReadFifoCache(backing io.ReadSeeker, header *PixiHeader, layer *Layer, maxSize int) *LayerReadFifoCache {
	return &LayerReadFifoCache{
		backing: backing,
		header:  header,
		layer:   layer,
		cache:   make(map[int]LayerFifoCacheTile),
		maxSize: maxSize,
	}
}

func (c *LayerReadFifoCache) Layer() *Layer {
	return c.layer
}

func (c *LayerReadFifoCache) Header() *PixiHeader {
	return c.header
}

func (c *LayerReadFifoCache) Get(tile int) ([]byte, error) {
	c.cacheLock.RLock()
	cached, found := c.cache[tile]
	c.cacheLock.RUnlock()
	if found {
		return cached.data, nil
	}

	data := make([]byte, c.layer.DiskTileSize(tile))
	c.cacheLock.Lock()
	err := c.layer.ReadTile(c.backing, c.header, tile, data)
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

func NewLayerFifoCache(backing io.ReadWriteSeeker, header *PixiHeader, layer *Layer, maxSize int) *LayerFifoCache {
	return &LayerFifoCache{
		LayerReadFifoCache: LayerReadFifoCache{
			backing: backing,
			header:  header,
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
		tileData := make([]byte, c.layer.DiskTileSize(tile))
		err := c.layer.ReadTile(c.backing, c.header, tile, tileData)
		if err != nil {
			return err
		}
		copy(tileData[tileOffset:], data)

		return c.evictAndAdd(tile, tileData)
	}

	return nil
}

func (c *LayerFifoCache) SetBit(tile int, bitIndex int, value bool) error {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	cached, found := c.cache[tile]
	if found {
		PackBool(value, cached.data, bitIndex)
		c.cache[tile] = LayerFifoCacheTile{
			age:  time.Now(),
			data: cached.data,
		}
	} else {
		tileData := make([]byte, c.layer.DiskTileSize(tile))
		err := c.layer.ReadTile(c.backing, c.header, tile, tileData)
		if err != nil {
			return err
		}
		PackBool(value, tileData, bitIndex)

		return c.evictAndAdd(tile, tileData)
	}

	return nil
}

func (c *LayerFifoCache) evictAndAdd(tile int, data []byte) error {
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
		err := c.layer.OverwriteTile(c.backing, c.header, oldestTile, oldest.data)
		if err != nil {
			return err
		}
		delete(c.cache, oldestTile)
	}
	c.cache[tile] = LayerFifoCacheTile{
		age:  time.Now(),
		data: data,
	}
	return nil
}

func (c *LayerFifoCache) Flush() error {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	for tile, entry := range c.cache {
		err := c.layer.OverwriteTile(c.backing, c.header, tile, entry.data)
		if err != nil {
			return err
		}
	}
	return nil
}

type ReadCachedLayer struct {
	cache CachedLayerReadCache
}

type WriteCachedLayer struct {
	cache CachedLayerCache
}

type CachedLayer struct {
	ReadCachedLayer
	WriteCachedLayer
}

func NewReadCachedLayer(cache CachedLayerReadCache) *ReadCachedLayer {
	return &ReadCachedLayer{
		cache: cache,
	}
}

func NewCachedLayer(cache CachedLayerCache) *CachedLayer {
	return &CachedLayer{
		ReadCachedLayer:  ReadCachedLayer{cache: cache},
		WriteCachedLayer: WriteCachedLayer{cache: cache},
	}
}

func (s *ReadCachedLayer) Layer() *Layer {
	return s.cache.Layer()
}

func (s *WriteCachedLayer) Layer() *Layer {
	return s.cache.Layer()
}

func (s *CachedLayer) Layer() *Layer {
	return s.ReadCachedLayer.cache.Layer()
}

func (s *ReadCachedLayer) SampleAt(coord SampleCoordinate) (Sample, error) {
	tileSelector := coord.ToTileSelector(s.Layer().Dimensions)
	sample := make([]any, len(s.Layer().Fields))

	if s.Layer().Separated {
		for fieldIndex, field := range s.Layer().Fields {
			fieldTile := tileSelector.Tile + s.Layer().Dimensions.Tiles()*fieldIndex

			tileData, err := s.cache.Get(fieldTile)
			if err != nil {
				return nil, err
			}

			if field.Type == FieldBool {
				sample[fieldIndex] = UnpackBool(tileData, tileSelector.InTile)
			} else {
				fieldOffset := tileSelector.InTile * field.Size()
				sample[fieldIndex] = field.BytesToValue(tileData[fieldOffset:], s.cache.Header().ByteOrder)
			}
		}
	} else {
		fieldOffset := tileSelector.InTile * s.Layer().Fields.Size()

		tileData, err := s.cache.Get(tileSelector.Tile)
		if err != nil {
			return nil, err
		}
		for i, field := range s.Layer().Fields {
			sample[i] = field.BytesToValue(tileData[fieldOffset:], s.cache.Header().ByteOrder)
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

		tileData, err := s.cache.Get(fieldTile)
		if err != nil {
			return nil, err
		}

		if field.Type == FieldBool {
			return UnpackBool(tileData, tileSelector.InTile), nil
		} else {
			fieldOffset := tileSelector.InTile * field.Size()
			return field.BytesToValue(tileData[fieldOffset:], s.cache.Header().ByteOrder), nil
		}
	} else {
		tileData, err := s.cache.Get(tileSelector.Tile)
		if err != nil {
			return nil, err
		}
		fieldOffset := tileSelector.InTile * s.Layer().Fields.Size()
		for _, field := range s.Layer().Fields[:fieldIndex] {
			fieldOffset += field.Size()
		}
		return field.BytesToValue(tileData[fieldOffset:], s.cache.Header().ByteOrder), nil
	}
}

func (s *WriteCachedLayer) SetSampleAt(coord SampleCoordinate, values Sample) error {
	if s.Layer().Compression != CompressionNone {
		panic("pixi: cannot set direct access sample on compressed layer")
	}
	if len(values) != len(s.Layer().Fields) {
		panic("pixi: values length does not match field count")
	}

	tileSelector := coord.ToTileSelector(s.Layer().Dimensions)

	if s.Layer().Separated {
		for fieldIndex, field := range s.Layer().Fields {
			separatedTileIndex := tileSelector.Tile + s.Layer().Dimensions.Tiles()*fieldIndex

			if field.Type == FieldBool {
				err := s.cache.SetBit(separatedTileIndex, tileSelector.InTile, values[fieldIndex].(bool))
				if err != nil {
					return err
				}
			} else {
				fieldInTileOffset := tileSelector.InTile * field.Size()
				raw := make([]byte, field.Size())
				field.ValueToBytes(values[fieldIndex], s.cache.Header().ByteOrder, raw)
				err := s.cache.SetFragment(separatedTileIndex, fieldInTileOffset, raw)
				if err != nil {
					return err
				}
			}
		}
	} else {
		fieldInTileOffset := tileSelector.InTile * s.Layer().Fields.Size()
		raw := make([]byte, s.Layer().Fields.Size())
		fieldOffset := 0
		for i, field := range s.Layer().Fields {
			field.ValueToBytes(values[i], s.cache.Header().ByteOrder, raw[fieldOffset:])
			fieldOffset += field.Size()
		}
		return s.cache.SetFragment(tileSelector.Tile, fieldInTileOffset, raw)
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

	if s.Layer().Separated {
		separatedTileIndex := tileSelector.Tile + s.Layer().Dimensions.Tiles()*fieldIndex

		if field.Type == FieldBool {
			return s.cache.SetBit(separatedTileIndex, tileSelector.InTile, value.(bool))
		} else {
			fieldInTileOffset := tileSelector.InTile * field.Size()
			raw := make([]byte, field.Size())
			field.ValueToBytes(value, s.cache.Header().ByteOrder, raw)
			return s.cache.SetFragment(separatedTileIndex, fieldInTileOffset, raw)
		}
	} else {
		fieldTileOffset := tileSelector.InTile * s.Layer().Fields.Size()
		for _, field := range s.Layer().Fields[:fieldIndex] {
			fieldTileOffset += field.Size()
		}
		raw := make([]byte, field.Size())
		field.ValueToBytes(value, s.cache.Header().ByteOrder, raw)
		return s.cache.SetFragment(tileSelector.Tile, fieldTileOffset, raw)
	}
}

func (s *WriteCachedLayer) Flush() error {
	return s.cache.Flush()
}
