package read

import (
	"io"
	"math"
	"sync"

	"github.com/owlpinetech/pixi"
)

type CacheManager[K comparable, V any] interface {
	MaxInCache() int
	Add(key K, value V, cache *sync.Map)
	Access(key K)
}

type LayerReadCache struct {
	lock    sync.RWMutex
	layer   *pixi.Layer
	header  *pixi.PixiHeader
	backing io.ReadSeeker
	cache   *sync.Map // map[int][]byte, but safe for concurrent access/modification
	manager CacheManager[int, []byte]
}

func NewLayerReadCache(backing io.ReadSeeker, header *pixi.PixiHeader, layer *pixi.Layer, eviction CacheManager[int, []byte]) *LayerReadCache {
	return &LayerReadCache{
		header:  header,
		layer:   layer,
		backing: backing,
		cache:   &sync.Map{},
		manager: eviction,
	}
}

func (c *LayerReadCache) SampleAt(coord pixi.SampleCoordinate) ([]any, error) {
	tileSelector := coord.ToTileSelector(c.layer.Dimensions)
	sample := make([]any, len(c.layer.Fields))
	if c.layer.Separated {
		for fieldIndex, field := range c.layer.Fields {
			fieldTile := tileSelector.Tile + c.layer.Dimensions.Tiles()*fieldIndex
			fieldOffset := tileSelector.InTile * field.Size()

			tileData, err := c.getTile(fieldTile)
			if err != nil {
				return nil, err
			}

			sample[fieldIndex] = field.BytesToValue(tileData[fieldOffset:], c.header.ByteOrder)
		}
		return sample, nil
	} else {
		fieldOffset := tileSelector.InTile * c.layer.Fields.Size()

		tileData, err := c.getTile(tileSelector.Tile)
		if err != nil {
			return nil, err
		}
		for i, field := range c.layer.Fields {
			sample[i] = field.BytesToValue(tileData[fieldOffset:], c.header.ByteOrder)
			fieldOffset += field.Size()
		}
		return sample, nil
	}
}

func (c *LayerReadCache) FieldAt(coord pixi.SampleCoordinate, fieldIndex int) (any, error) {
	tileSelector := coord.ToTileSelector(c.layer.Dimensions)
	offset := tileSelector.InTile
	if c.layer.Separated {
		tileSelector.Tile *= c.layer.Dimensions.Tiles()
		offset *= c.layer.Fields[fieldIndex].Size()
	} else {
		offset *= c.layer.Fields.Size()
		for _, field := range c.layer.Fields[:fieldIndex] {
			offset += field.Size()
		}
	}

	tileData, err := c.getTile(tileSelector.Tile)
	if err != nil {
		return nil, err
	}
	return c.layer.Fields[fieldIndex].BytesToValue(tileData[offset:], c.header.ByteOrder), nil
}

func (c *LayerReadCache) getTile(tileIndex int) ([]byte, error) {
	c.manager.Access(tileIndex)
	if tile, ok := c.cache.Load(tileIndex); ok {
		return tile.([]byte), nil
	} else {
		return c.loadTile(tileIndex)
	}
}

func (c *LayerReadCache) loadTile(tileIndex int) ([]byte, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	// in case multiple readers get locked trying to load the same tile
	if tile, ok := c.cache.Load(tileIndex); ok {
		return tile.([]byte), nil
	}

	chunk := make([]byte, c.layer.DiskTileSize(tileIndex))
	err := c.layer.ReadTile(c.backing, c.header, tileIndex, chunk)
	if err != nil {
		return nil, err
	}
	c.manager.Add(tileIndex, chunk, c.cache)
	tileData, _ := c.cache.Load(tileIndex)
	return tileData.([]byte), nil
}

type LfuCacheManager struct {
	lock       sync.RWMutex
	maxInCache int
	stored     int
	usages     *sync.Map
}

func NewLfuCacheManager(maxInCache int) *LfuCacheManager {
	return &LfuCacheManager{maxInCache: maxInCache, usages: &sync.Map{}}
}

func (m *LfuCacheManager) Access(key int) {
	if val, ok := m.usages.Load(key); ok {
		m.usages.Store(key, val.(int)+1)
	}
}

func (m *LfuCacheManager) Add(key int, value []byte, cache *sync.Map) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.stored >= m.maxInCache {
		// find least recently used to evict
		min := math.MaxInt
		toEvict := -1
		m.usages.Range(func(key any, val any) bool {
			if val.(int) < min {
				min = val.(int)
				toEvict = key.(int)
			}
			return true
		})
		m.usages.Delete(toEvict)
		cache.Delete(toEvict)
		m.stored -= 1
	}
	m.stored += 1
	m.usages.Store(key, 1)
	cache.Store(key, value)
}

func (m *LfuCacheManager) MaxInCache() int {
	return m.maxInCache
}
