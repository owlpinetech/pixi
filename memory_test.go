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

	sample0, err := SampleAt(stored, SampleCoordinate{25, 25})
	if err != nil {
		t.Fatal(err)
	}
	if sample0[0] != uint16(0) || sample0[1] != uint32(0) {
		t.Fatalf("expected initial sample to be all zero, got %v", sample0)
	}

	err = SetSampleAt(stored, SampleCoordinate{25, 25}, []any{uint16(42), uint32(4242)})
	if err != nil {
		t.Fatal(err)
	}

	sample1, err := SampleAt(stored, SampleCoordinate{25, 25})
	if err != nil {
		t.Fatal(err)
	}
	if sample1[0] != uint16(42) || sample1[1] != uint32(4242) {
		t.Fatalf("expected sample to be set to [42 4242], got %v", sample1)
	}
}

func TestMemoryLayerMinMaxTracking(t *testing.T) {
	headers := allHeaderVariants(Version)

	for _, h := range headers {
		buf := buffer.NewBuffer(1000)

		// Create a layer with multiple fields
		layer := &Layer{
			Name:        "test",
			Separated:   false,
			Compression: CompressionNone,
			Dimensions: DimensionSet{
				{Name: "x", Size: 2, TileSize: 2},
				{Name: "y", Size: 2, TileSize: 2},
			},
			Fields: FieldSet{
				{Name: "temperature", Type: FieldFloat32},
				{Name: "count", Type: FieldInt16},
			},
		}
		layer.TileBytes = make([]int64, layer.Dimensions.Tiles())
		layer.TileOffsets = make([]int64, layer.Dimensions.Tiles())

		memLayer := NewMemoryLayer(buf, h, layer)

		// Test SetFieldAt
		testData := []struct {
			coord SampleCoordinate
			temp  float32
			count int16
		}{
			{SampleCoordinate{0, 0}, 25.5, 10},
			{SampleCoordinate{1, 0}, -5.2, 25},
			{SampleCoordinate{0, 1}, 35.8, 5},
			{SampleCoordinate{1, 1}, 15.0, 30},
		}

		for _, data := range testData {
			err := SetFieldAt(memLayer, data.coord, 0, data.temp)
			if err != nil {
				t.Fatalf("SetFieldAt failed: %v", err)
			}
			err = SetFieldAt(memLayer, data.coord, 1, data.count)
			if err != nil {
				t.Fatalf("SetFieldAt failed: %v", err)
			}
		}

		// Check Min/Max for temperature field
		if layer.Fields[0].Min == nil || layer.Fields[0].Min.(float32) != -5.2 {
			t.Errorf("Expected temperature min to be -5.2, got %v", layer.Fields[0].Min)
		}
		if layer.Fields[0].Max == nil || layer.Fields[0].Max.(float32) != 35.8 {
			t.Errorf("Expected temperature max to be 35.8, got %v", layer.Fields[0].Max)
		}

		// Check Min/Max for count field
		if layer.Fields[1].Min == nil || layer.Fields[1].Min.(int16) != 5 {
			t.Errorf("Expected count min to be 5, got %v", layer.Fields[1].Min)
		}
		if layer.Fields[1].Max == nil || layer.Fields[1].Max.(int16) != 30 {
			t.Errorf("Expected count max to be 30, got %v", layer.Fields[1].Max)
		}

		// Test SetSampleAt
		err := SetSampleAt(memLayer, SampleCoordinate{0, 0}, []any{float32(-10.5), int16(2)})
		if err != nil {
			t.Fatalf("SetSampleAt failed: %v", err)
		}

		// Check that Min was updated
		if layer.Fields[0].Min == nil || layer.Fields[0].Min.(float32) != -10.5 {
			t.Errorf("Expected updated temperature min to be -10.5, got %v", layer.Fields[0].Min)
		}
		if layer.Fields[1].Min == nil || layer.Fields[1].Min.(int16) != 2 {
			t.Errorf("Expected updated count min to be 2, got %v", layer.Fields[1].Min)
		}
	}
}
