package pixi

import (
	"encoding/binary"
	"testing"

	"github.com/owlpinetech/pixi/internal/buffer"
)

func TestMemoryLayerBooleanFields(t *testing.T) {
	header := &PixiHeader{
		Version:    Version,
		ByteOrder:  binary.BigEndian,
		OffsetSize: 8,
	}

	// Test both separated and contiguous modes
	modes := []struct {
		name      string
		separated bool
	}{
		{"contiguous", false},
		{"separated", true},
	}

	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			// Create layer with boolean and other fields
			fields := FieldSet{
				{Name: "active", Type: FieldBool},
				{Name: "count", Type: FieldInt32},
				{Name: "enabled", Type: FieldBool},
			}
			dimensions := DimensionSet{{Name: "x", Size: 10, TileSize: 5}}

			layer := NewLayer("test", mode.separated, CompressionNone, dimensions, fields)

			// Create some test data
			testData := []struct {
				active  bool
				count   int32
				enabled bool
			}{
				{true, 100, false},
				{false, 200, true},
				{true, 300, true},
				{false, 400, false},
				{true, 500, true},
			}

			// Write initial data to create tiles
			wrtBuf := buffer.NewBuffer(100)
			for tileIndex := 0; tileIndex < layer.DiskTiles(); tileIndex++ {
				tileData := make([]byte, layer.DiskTileSize(tileIndex))
				layer.WriteTile(wrtBuf, header, tileIndex, tileData)
			}

			// Create memory layer
			rdBuf := buffer.NewBufferFrom(wrtBuf.Bytes())
			memLayer := NewMemoryLayer(rdBuf, header, layer)

			// Test SampleAt and SetSampleAt
			t.Run("SampleAt_SetSampleAt", func(t *testing.T) {
				for i, data := range testData {
					coord := SampleCoordinate{i}
					sample := Sample{data.active, data.count, data.enabled}

					// Set the sample
					err := memLayer.SetSampleAt(coord, sample)
					if err != nil {
						t.Fatalf("SetSampleAt failed at coord %v: %v", coord, err)
					}

					// Read it back
					readSample, err := memLayer.SampleAt(coord)
					if err != nil {
						t.Fatalf("SampleAt failed at coord %v: %v", coord, err)
					}

					// Verify each field
					if readSample[0].(bool) != data.active {
						t.Errorf("Active field mismatch at %v: expected %v, got %v", coord, data.active, readSample[0])
					}
					if readSample[1].(int32) != data.count {
						t.Errorf("Count field mismatch at %v: expected %v, got %v", coord, data.count, readSample[1])
					}
					if readSample[2].(bool) != data.enabled {
						t.Errorf("Enabled field mismatch at %v: expected %v, got %v", coord, data.enabled, readSample[2])
					}
				}
			})

			// Test FieldAt and SetFieldAt
			t.Run("FieldAt_SetFieldAt", func(t *testing.T) {
				for i, data := range testData {
					coord := SampleCoordinate{i}

					// Set individual fields
					err := memLayer.SetFieldAt(coord, 0, data.active)
					if err != nil {
						t.Fatalf("SetFieldAt(active) failed at coord %v: %v", coord, err)
					}
					err = memLayer.SetFieldAt(coord, 1, data.count)
					if err != nil {
						t.Fatalf("SetFieldAt(count) failed at coord %v: %v", coord, err)
					}
					err = memLayer.SetFieldAt(coord, 2, data.enabled)
					if err != nil {
						t.Fatalf("SetFieldAt(enabled) failed at coord %v: %v", coord, err)
					}

					// Read individual fields
					activeVal, err := memLayer.FieldAt(coord, 0)
					if err != nil {
						t.Fatalf("FieldAt(active) failed at coord %v: %v", coord, err)
					}
					countVal, err := memLayer.FieldAt(coord, 1)
					if err != nil {
						t.Fatalf("FieldAt(count) failed at coord %v: %v", coord, err)
					}
					enabledVal, err := memLayer.FieldAt(coord, 2)
					if err != nil {
						t.Fatalf("FieldAt(enabled) failed at coord %v: %v", coord, err)
					}

					// Verify values
					if activeVal.(bool) != data.active {
						t.Errorf("FieldAt(active) mismatch at %v: expected %v, got %v", coord, data.active, activeVal)
					}
					if countVal.(int32) != data.count {
						t.Errorf("FieldAt(count) mismatch at %v: expected %v, got %v", coord, data.count, countVal)
					}
					if enabledVal.(bool) != data.enabled {
						t.Errorf("FieldAt(enabled) mismatch at %v: expected %v, got %v", coord, data.enabled, enabledVal)
					}
				}
			})
		})
	}
}

func TestCachedLayerBooleanFields(t *testing.T) {
	header := &PixiHeader{
		Version:    Version,
		ByteOrder:  binary.BigEndian,
		OffsetSize: 8,
	}

	// Test both separated and contiguous modes
	modes := []struct {
		name      string
		separated bool
	}{
		{"contiguous", false},
		{"separated", true},
	}

	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			// Create layer with boolean and other fields
			fields := FieldSet{
				{Name: "visible", Type: FieldBool},
				{Name: "priority", Type: FieldInt16},
				{Name: "active", Type: FieldBool},
			}
			dimensions := DimensionSet{{Name: "x", Size: 8, TileSize: 4}}

			layer := NewLayer("test", mode.separated, CompressionNone, dimensions, fields)

			// Create test data
			testData := []struct {
				visible  bool
				priority int16
				active   bool
			}{
				{true, 100, false},
				{false, 200, true},
				{true, 300, true},
				{false, 400, false},
			}

			// Write initial data to create tiles
			wrtBuf := buffer.NewBuffer(100)
			for tileIndex := 0; tileIndex < layer.DiskTiles(); tileIndex++ {
				tileData := make([]byte, layer.DiskTileSize(tileIndex))
				layer.WriteTile(wrtBuf, header, tileIndex, tileData)
			}

			// Create cached layer
			rdBuf := buffer.NewBufferFrom(wrtBuf.Bytes())
			cache := NewCachedLayer(NewLayerFifoCache(rdBuf, header, layer, 100))

			// Test SampleAt and SetSampleAt
			t.Run("SampleAt_SetSampleAt", func(t *testing.T) {
				for i, data := range testData {
					coord := SampleCoordinate{i}
					sample := Sample{data.visible, data.priority, data.active}

					// Set the sample
					err := cache.SetSampleAt(coord, sample)
					if err != nil {
						t.Fatalf("SetSampleAt failed at coord %v: %v", coord, err)
					}

					// Read it back
					readSample, err := cache.SampleAt(coord)
					if err != nil {
						t.Fatalf("SampleAt failed at coord %v: %v", coord, err)
					}

					// Verify each field
					if readSample[0].(bool) != data.visible {
						t.Errorf("Visible field mismatch at %v: expected %v, got %v", coord, data.visible, readSample[0])
					}
					if readSample[1].(int16) != data.priority {
						t.Errorf("Priority field mismatch at %v: expected %v, got %v", coord, data.priority, readSample[1])
					}
					if readSample[2].(bool) != data.active {
						t.Errorf("Active field mismatch at %v: expected %v, got %v", coord, data.active, readSample[2])
					}
				}
			})

			// Test FieldAt and SetFieldAt
			t.Run("FieldAt_SetFieldAt", func(t *testing.T) {
				for i, data := range testData {
					coord := SampleCoordinate{i}

					// Set individual fields
					err := cache.SetFieldAt(coord, 0, data.visible)
					if err != nil {
						t.Fatalf("SetFieldAt(visible) failed at coord %v: %v", coord, err)
					}
					err = cache.SetFieldAt(coord, 1, data.priority)
					if err != nil {
						t.Fatalf("SetFieldAt(priority) failed at coord %v: %v", coord, err)
					}
					err = cache.SetFieldAt(coord, 2, data.active)
					if err != nil {
						t.Fatalf("SetFieldAt(active) failed at coord %v: %v", coord, err)
					}

					// Read individual fields
					visibleVal, err := cache.FieldAt(coord, 0)
					if err != nil {
						t.Fatalf("FieldAt(visible) failed at coord %v: %v", coord, err)
					}
					priorityVal, err := cache.FieldAt(coord, 1)
					if err != nil {
						t.Fatalf("FieldAt(priority) failed at coord %v: %v", coord, err)
					}
					activeVal, err := cache.FieldAt(coord, 2)
					if err != nil {
						t.Fatalf("FieldAt(active) failed at coord %v: %v", coord, err)
					}

					// Verify values
					if visibleVal.(bool) != data.visible {
						t.Errorf("FieldAt(visible) mismatch at %v: expected %v, got %v", coord, data.visible, visibleVal)
					}
					if priorityVal.(int16) != data.priority {
						t.Errorf("FieldAt(priority) mismatch at %v: expected %v, got %v", coord, data.priority, priorityVal)
					}
					if activeVal.(bool) != data.active {
						t.Errorf("FieldAt(active) mismatch at %v: expected %v, got %v", coord, data.active, activeVal)
					}
				}
			})
		})
	}
}

func TestBooleanFieldBitPacking(t *testing.T) {
	header := &PixiHeader{
		Version:    Version,
		ByteOrder:  binary.BigEndian,
		OffsetSize: 8,
	}

	// Test specifically the bitfield packing in separated mode
	fields := FieldSet{
		{Name: "flags", Type: FieldBool},
	}
	// Use 10 samples to test byte boundary (8 bits + 2 bits)
	dimensions := DimensionSet{{Name: "x", Size: 10, TileSize: 10}}

	layer := NewLayer("test", true, CompressionNone, dimensions, fields) // separated = true

	// Verify tile size is correctly calculated for bitfield
	expectedTileSize := (10 + 7) / 8 // 10 bits = 2 bytes
	actualTileSize := layer.DiskTileSize(0)
	if actualTileSize != expectedTileSize {
		t.Errorf("DiskTileSize for boolean bitfield: expected %d bytes, got %d", expectedTileSize, actualTileSize)
	}

	// Write initial data
	wrtBuf := buffer.NewBuffer(100)
	tileData := make([]byte, layer.DiskTileSize(0))
	layer.WriteTile(wrtBuf, header, 0, tileData)

	// Create memory layer and test specific bit patterns
	rdBuf := buffer.NewBufferFrom(wrtBuf.Bytes())
	memLayer := NewMemoryLayer(rdBuf, header, layer)

	// Test pattern: alternating true/false
	testPattern := []bool{true, false, true, false, true, false, true, false, true, false}

	// Set the pattern
	for i, value := range testPattern {
		coord := SampleCoordinate{i}
		err := memLayer.SetFieldAt(coord, 0, value)
		if err != nil {
			t.Fatalf("SetFieldAt failed at coord %d: %v", i, err)
		}
	}

	// Read back and verify
	for i, expected := range testPattern {
		coord := SampleCoordinate{i}
		value, err := memLayer.FieldAt(coord, 0)
		if err != nil {
			t.Fatalf("FieldAt failed at coord %d: %v", i, err)
		}
		if value.(bool) != expected {
			t.Errorf("Bit pattern mismatch at position %d: expected %v, got %v", i, expected, value)
		}
	}
}
