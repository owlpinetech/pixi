package pixi

import (
	"encoding/binary"
	"math/rand/v2"
	"sync"
	"testing"

	"github.com/owlpinetech/pixi/internal/buffer"
)

func testSampleAtSameAsRaw(t *testing.T, wg *sync.WaitGroup, layer TileAccessLayer, coord SampleCoordinate, expect any) {
	defer wg.Done()
	at, err := ChannelAt(layer, coord, 1)
	if err != nil {
		t.Error(err)
	} else if at != expect {
		t.Errorf("expected %v but got %v at coord %v", expect, at, coord)
	}
}

func TestCachedSampleChannelConcurrent(t *testing.T) {
	header := &Header{
		Version:    Version,
		OffsetSize: 4,
		ByteOrder:  binary.BigEndian,
	}
	layer := NewLayer(
		"concurrent-test",
		DimensionSet{{Name: "x", Size: 50, TileSize: 10}, {Name: "y", Size: 50, TileSize: 10}},
		ChannelSet{{Name: "one", Type: ChannelUint16}, {Name: "two", Type: ChannelUint32}},
	)

	// write some test data
	wrtBuf := buffer.NewBuffer(10)
	rawTiles := [][]byte{}
	for i := range layer.Dimensions.Tiles() {
		chunk := make([]byte, layer.DiskTileSize(i))
		for i := range chunk {
			chunk[i] = byte(rand.IntN(256))
		}
		layer.WriteTile(wrtBuf, header, i, chunk)
		rawTiles = append(rawTiles, chunk)
	}

	// create a cache
	rdBuffer := buffer.NewBufferFrom(wrtBuf.Bytes())
	cache := NewFifoCacheLayer(rdBuffer, header, layer, 4)

	// we're only going to look at the second channel for this test
	testSampleCount := layer.Dimensions.Samples() / 4
	testCoords := make([]SampleCoordinate, testSampleCount)
	testExpect := make([]any, testSampleCount) // offset into raw tile chunk, not written data
	for i := range testCoords {
		testIndex := SampleIndex(rand.IntN(layer.Dimensions.Samples()))
		testTile := testIndex.ToSampleCoordinate(layer.Dimensions).ToTileSelector(layer.Dimensions)
		testCoords[i] = testIndex.ToSampleCoordinate(layer.Dimensions)
		testExpect[i] = layer.Channels[1].Value(rawTiles[testTile.Tile][testTile.InTile*layer.Channels.Size()+layer.Channels[0].Size():], header.ByteOrder)
	}

	var wg sync.WaitGroup
	for randInd := range testCoords {
		wg.Add(1)
		go testSampleAtSameAsRaw(t, &wg, cache, testCoords[randInd], testExpect[randInd])
	}
	wg.Wait()
}

func TestCachedSetSampleAt(t *testing.T) {
	header := &Header{
		Version:    Version,
		OffsetSize: 4,
		ByteOrder:  binary.BigEndian,
	}
	wrtBuf := buffer.NewBuffer(10)
	header.WriteHeader(wrtBuf)

	layer, err := newBlankUncompressedLayer(
		wrtBuf,
		header,
		"stored-set-sample-at",
		DimensionSet{{Name: "x", Size: 50, TileSize: 10}, {Name: "y", Size: 50, TileSize: 10}},
		ChannelSet{{Name: "one", Type: ChannelUint16}, {Name: "two", Type: ChannelUint32}},
	)
	if err != nil {
		t.Fatal(err)
	}

	cached := NewFifoCacheLayer(wrtBuf, header, layer, 4)

	sample0, err := SampleAt(cached, SampleCoordinate{25, 25})
	if err != nil {
		t.Fatal(err)
	}
	if sample0[0] != uint16(0) || sample0[1] != uint32(0) {
		t.Fatalf("expected initial sample to be all zero, got %v", sample0)
	}

	err = SetSampleAt(cached, SampleCoordinate{25, 25}, []any{uint16(42), uint32(4242)})
	if err != nil {
		t.Fatal(err)
	}

	sample1, err := SampleAt(cached, SampleCoordinate{25, 25})
	if err != nil {
		t.Fatal(err)
	}
	if sample1[0] != uint16(42) || sample1[1] != uint32(4242) {
		t.Fatalf("expected sample to be set to [42 4242], got %v", sample1)
	}
}
