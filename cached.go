package pixi

import (
	"io"
	"sync"
	"time"
)

type FifoCacheLayerTile struct {
	age  time.Time
	data []byte
}

type FifoCacheReadLayer struct {
	cacheLock sync.RWMutex
	backing   io.ReadSeeker
	header    Header
	layer     Layer
	cache     map[int]FifoCacheLayerTile
	maxSize   int
}

// Compile-time check to ensure LayerReadFifoCache implements TileAccessLayer
var _ TileAccessLayer = (*FifoCacheReadLayer)(nil)

func NewFifoCacheReadLayer(backing io.ReadSeeker, header Header, layer Layer, maxSize int) *FifoCacheReadLayer {
	return &FifoCacheReadLayer{
		backing: backing,
		header:  header,
		layer:   layer,
		cache:   make(map[int]FifoCacheLayerTile),
		maxSize: maxSize,
	}
}

func (c *FifoCacheReadLayer) Layer() Layer {
	return c.layer
}

func (c *FifoCacheReadLayer) Header() Header {
	return c.header
}

func (c *FifoCacheReadLayer) Tile(tile int) ([]byte, error) {
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
	c.cache[tile] = FifoCacheLayerTile{
		age:  time.Now(),
		data: data,
	}
	c.cacheLock.Unlock()

	return data, nil
}

type FifoCacheLayer struct {
	FifoCacheReadLayer
	backing io.ReadWriteSeeker
}

// Compile-time check to ensure LayerFifoCache implements CachedLayerCache
var _ TileModifierLayer = (*FifoCacheLayer)(nil)

func NewFifoCacheLayer(backing io.ReadWriteSeeker, header Header, layer Layer, maxSize int) *FifoCacheLayer {
	return &FifoCacheLayer{
		FifoCacheReadLayer: FifoCacheReadLayer{
			backing: backing,
			header:  header,
			layer:   layer,
			cache:   make(map[int]FifoCacheLayerTile),
			maxSize: maxSize,
		},
		backing: backing,
	}
}

func (c *FifoCacheLayer) SetDirty(tile int) {
	// no-op for fifo cache
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	cached, found := c.cache[tile]
	if found {
		c.cache[tile] = FifoCacheLayerTile{
			age:  time.Now(),
			data: cached.data,
		}
	}
}

func (c *FifoCacheLayer) Commit() error {
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
