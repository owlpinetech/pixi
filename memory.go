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

var _ TileAccessLayer = (*MemoryLayer)(nil)
var _ TileModifierLayer = (*MemoryLayer)(nil)

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

func (s *MemoryLayer) Tile(tile int) ([]byte, error) {
	return s.loadTile(tile)
}

func (s *MemoryLayer) SetDirty(tile int) {
	// no-op for memory layer
}

func (s *MemoryLayer) Commit() error {
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
