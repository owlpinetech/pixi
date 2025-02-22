package read

import (
	"encoding/binary"
	"math/rand/v2"
	"sync"
	"testing"

	"github.com/owlpinetech/pixi"
	"github.com/owlpinetech/pixi/internal/buffer"
)

func TestCacheSampleFieldConcurrent(t *testing.T) {
	header := pixi.PixiHeader{
		Version:    pixi.Version,
		OffsetSize: 4,
		ByteOrder:  binary.BigEndian,
	}
	layer := pixi.NewLayer(
		"concurrent-test",
		false,
		pixi.CompressionNone,
		pixi.DimensionSet{{Name: "x", Size: 500, TileSize: 100}, {Name: "y", Size: 500, TileSize: 100}},
		[]pixi.Field{{Name: "one", Type: pixi.FieldUint16}, {Name: "two", Type: pixi.FieldUint32}},
	)

	// write some test data
	wrtBuf := buffer.NewBuffer(10)
	rawTiles := [][]byte{}
	for i := range layer.Dimensions.Tiles() {
		chunk := make([]byte, layer.DiskTileSize(i))
		for i := 0; i < len(chunk); i++ {
			chunk[i] = byte(rand.IntN(256))
		}
		layer.WriteTile(wrtBuf, header, i, chunk)
		rawTiles = append(rawTiles, chunk)
	}

	// create a cache
	rdBuffer := buffer.NewBufferFrom(wrtBuf.Bytes())
	cache := NewLayerReadCache(rdBuffer, header, layer, NewLfuCacheManager(4))

	// we're only going to look at the second field for this test
	testSampleCount := layer.Dimensions.Samples() / 4
	testCoords := make([]pixi.SampleCoordinate, testSampleCount)
	testExpect := make([]any, testSampleCount) // offset into raw tile chunk, not written data
	for i := range testCoords {
		testIndex := pixi.SampleIndex(rand.IntN(layer.Dimensions.Samples()))
		testTile := testIndex.ToSampleCoordinate(layer.Dimensions).ToTileSelector(layer.Dimensions)
		testCoords[i] = testIndex.ToSampleCoordinate(layer.Dimensions)
		testExpect[i] = layer.Fields[1].BytesToValue(rawTiles[testTile.Tile][testTile.InTile*layer.Fields.Size()+layer.Fields[0].Size():], header.ByteOrder)
	}

	var wg sync.WaitGroup
	for randInd := range testCoords {
		wg.Add(1)
		go testSampleAtSameAsRaw(t, &wg, cache, testCoords[randInd], testExpect[randInd])
	}
	wg.Wait()
}

func testSampleAtSameAsRaw(t *testing.T, wg *sync.WaitGroup, cache *LayerReadCache, coord pixi.SampleCoordinate, expect any) {
	defer wg.Done()
	at, err := cache.FieldAt(coord, 1)
	if err != nil {
		t.Error(err)
	} else if at != expect {
		t.Errorf("expected %v but got %v at coord %v", expect, at, coord)
	}
}
