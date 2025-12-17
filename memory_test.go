package pixi

import (
	"encoding/binary"
	"math/rand/v2"
	"sync"
	"testing"

	"github.com/owlpinetech/pixi/internal/buffer"
)

func TestMemorySampleFieldConcurrent(t *testing.T) {
	header := &PixiHeader{
		Version:    Version,
		OffsetSize: 4,
		ByteOrder:  binary.BigEndian,
	}
	layer := NewLayer(
		"concurrent-test",
		false,
		CompressionNone,
		DimensionSet{{Name: "x", Size: 50, TileSize: 10}, {Name: "y", Size: 50, TileSize: 10}},
		FieldSet{{Name: "one", Type: FieldUint16}, {Name: "two", Type: FieldUint32}},
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

	// create a disk-backed layer
	rdBuffer := buffer.NewBufferFrom(wrtBuf.Bytes())
	stored := NewMemoryLayer(rdBuffer, header, layer)

	// we're only going to look at the second field for this test
	testSampleCount := layer.Dimensions.Samples() / 4
	testCoords := make([]SampleCoordinate, testSampleCount)
	testExpect := make([]any, testSampleCount) // offset into raw tile chunk, not written data
	for i := range testCoords {
		testIndex := SampleIndex(rand.IntN(layer.Dimensions.Samples()))
		testTile := testIndex.ToSampleCoordinate(layer.Dimensions).ToTileSelector(layer.Dimensions)
		testCoords[i] = testIndex.ToSampleCoordinate(layer.Dimensions)
		testExpect[i] = layer.Fields[1].Value(rawTiles[testTile.Tile][testTile.InTile*layer.Fields.Size()+layer.Fields[0].Size():], header.ByteOrder)
	}

	var wg sync.WaitGroup
	for randInd := range testCoords {
		wg.Add(1)
		go testSampleAtSameAsRaw(t, &wg, stored, testCoords[randInd], testExpect[randInd])
	}
	wg.Wait()
}

func TestMemorySetSampleAt(t *testing.T) {
	header := &PixiHeader{
		Version:    Version,
		OffsetSize: 4,
		ByteOrder:  binary.BigEndian,
	}
	wrtBuf := buffer.NewBuffer(10)
	header.WriteHeader(wrtBuf)

	layer, err := NewBlankUncompressedLayer(
		wrtBuf,
		header,
		"stored-set-sample-at",
		false,
		DimensionSet{{Name: "x", Size: 50, TileSize: 10}, {Name: "y", Size: 50, TileSize: 10}},
		FieldSet{{Name: "one", Type: FieldUint16}, {Name: "two", Type: FieldUint32}},
	)
	if err != nil {
		t.Fatal(err)
	}

	stored := NewMemoryLayer(wrtBuf, header, layer)

	sample0, err := stored.SampleAt(SampleCoordinate{25, 25})
	if err != nil {
		t.Fatal(err)
	}
	if sample0[0] != uint16(0) || sample0[1] != uint32(0) {
		t.Fatalf("expected initial sample to be all zero, got %v", sample0)
	}

	err = stored.SetSampleAt(SampleCoordinate{25, 25}, []any{uint16(42), uint32(4242)})
	if err != nil {
		t.Fatal(err)
	}

	sample1, err := stored.SampleAt(SampleCoordinate{25, 25})
	if err != nil {
		t.Fatal(err)
	}
	if sample1[0] != uint16(42) || sample1[1] != uint32(4242) {
		t.Fatalf("expected sample to be set to [42 4242], got %v", sample1)
	}
}
