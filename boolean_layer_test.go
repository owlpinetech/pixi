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

func TestTileOrderReadIteratorBooleanFields(t *testing.T) {
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
				{Name: "enabled", Type: FieldBool},
				{Name: "score", Type: FieldInt32},
				{Name: "active", Type: FieldBool},
			}
			dimensions := DimensionSet{{Name: "x", Size: 12, TileSize: 4}}

			// Create test data
			testData := []struct {
				enabled bool
				score   int32
				active  bool
			}{
				{true, 100, false},
				{false, 200, true},
				{true, 300, true},
				{false, 400, false},
				{true, 500, true},
				{false, 600, false},
				{true, 700, true},
				{false, 800, false},
				{true, 900, true},
				{false, 1000, false},
				{true, 1100, true},
				{false, 1200, false},
			}

			// Create blank uncompressed layer
			wrtBuf := buffer.NewBuffer(100)
			layer, err := NewBlankUncompressedLayer(
				wrtBuf,
				header,
				"tile-order-read-iterator-boolean-test",
				mode.separated,
				dimensions,
				fields,
			)
			if err != nil {
				t.Fatal(err)
			}

			// Create memory layer and set test data
			memLayer := NewMemoryLayer(wrtBuf, header, layer)

			for i, data := range testData {
				coord := SampleCoordinate{i}
				sample := Sample{data.enabled, data.score, data.active}
				err := memLayer.SetSampleAt(coord, sample)
				if err != nil {
					t.Fatalf("SetSampleAt failed at coord %v: %v", coord, err)
				}
			}
			memLayer.Flush()

			// Create iterator and verify data
			rdBuf := buffer.NewBufferFrom(wrtBuf.Bytes())
			iterator := NewTileOrderReadIterator(rdBuf, header, layer)
			defer iterator.Done()

			sampleIndex := 0
			for iterator.Next() {
				if sampleIndex >= len(testData) {
					t.Errorf("Iterator returned more samples than expected")
					break
				}

				coord := iterator.Coordinate()
				sample := iterator.Sample()
				expectedData := testData[sampleIndex]

				// Test Sample() method
				if sample[0].(bool) != expectedData.enabled {
					t.Errorf("Sample() enabled field mismatch at coord %v: expected %v, got %v", coord, expectedData.enabled, sample[0])
				}
				if sample[1].(int32) != expectedData.score {
					t.Errorf("Sample() score field mismatch at coord %v: expected %v, got %v", coord, expectedData.score, sample[1])
				}
				if sample[2].(bool) != expectedData.active {
					t.Errorf("Sample() active field mismatch at coord %v: expected %v, got %v", coord, expectedData.active, sample[2])
				}

				// Test Field() method
				enabledVal := iterator.Field(0)
				scoreVal := iterator.Field(1)
				activeVal := iterator.Field(2)

				if enabledVal.(bool) != expectedData.enabled {
					t.Errorf("Field(0) enabled field mismatch at coord %v: expected %v, got %v", coord, expectedData.enabled, enabledVal)
				}
				if scoreVal.(int32) != expectedData.score {
					t.Errorf("Field(1) score field mismatch at coord %v: expected %v, got %v", coord, expectedData.score, scoreVal)
				}
				if activeVal.(bool) != expectedData.active {
					t.Errorf("Field(2) active field mismatch at coord %v: expected %v, got %v", coord, expectedData.active, activeVal)
				}

				sampleIndex++
			}

			if iterator.Error() != nil {
				t.Fatalf("Iterator encountered error: %v", iterator.Error())
			}

			if sampleIndex != len(testData) {
				t.Errorf("Iterator did not return all expected samples: got %d, expected %d", sampleIndex, len(testData))
			}
		})
	}
}

func TestTileOrderWriteIteratorBooleanFields(t *testing.T) {
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
				{Name: "weight", Type: FieldFloat32},
				{Name: "locked", Type: FieldBool},
			}
			dimensions := DimensionSet{{Name: "x", Size: 8, TileSize: 4}}

			layer := NewLayer("test", mode.separated, CompressionNone, dimensions, fields)

			// Create test data
			testData := []struct {
				visible bool
				weight  float32
				locked  bool
			}{
				{true, 1.5, false},
				{false, 2.7, true},
				{true, 3.14, true},
				{false, 4.0, false},
				{true, 5.25, true},
				{false, 6.8, false},
				{true, 7.99, true},
				{false, 8.1, false},
			}

			// Create write iterator and write test data
			wrtBuf := buffer.NewBuffer(100)
			iterator := NewTileOrderWriteIterator(wrtBuf, header, layer)

			sampleIndex := 0
			for iterator.Next() {
				if sampleIndex >= len(testData) {
					t.Errorf("Iterator requested more samples than expected")
					break
				}

				coord := iterator.Coordinate()
				expectedData := testData[sampleIndex]

				// Test SetSample() method
				sample := Sample{expectedData.visible, expectedData.weight, expectedData.locked}
				iterator.SetSample(sample)

				if coord[0] != sampleIndex {
					t.Errorf("Unexpected coordinate at sample %d: got %v, expected {%d}", sampleIndex, coord, sampleIndex)
				}

				sampleIndex++
			}

			iterator.Done()

			if iterator.Error() != nil {
				t.Fatalf("Write iterator encountered error: %v", iterator.Error())
			}

			if sampleIndex != len(testData) {
				t.Errorf("Write iterator did not process all expected samples: got %d, expected %d", sampleIndex, len(testData))
			}

			// Verify written data using memory layer
			rdBuf := buffer.NewBufferFrom(wrtBuf.Bytes())
			memLayer := NewMemoryLayer(rdBuf, header, layer)

			for i, expectedData := range testData {
				coord := SampleCoordinate{i}
				sample, err := memLayer.SampleAt(coord)
				if err != nil {
					t.Fatalf("SampleAt failed at coord %v: %v", coord, err)
				}

				if sample[0].(bool) != expectedData.visible {
					t.Errorf("Visible field mismatch at coord %v: expected %v, got %v", coord, expectedData.visible, sample[0])
				}
				if sample[1].(float32) != expectedData.weight {
					t.Errorf("Weight field mismatch at coord %v: expected %v, got %v", coord, expectedData.weight, sample[1])
				}
				if sample[2].(bool) != expectedData.locked {
					t.Errorf("Locked field mismatch at coord %v: expected %v, got %v", coord, expectedData.locked, sample[2])
				}
			}
		})
	}

	// Test SetField() method specifically
	t.Run("SetField", func(t *testing.T) {
		fields := FieldSet{
			{Name: "flag1", Type: FieldBool},
			{Name: "value", Type: FieldInt16},
			{Name: "flag2", Type: FieldBool},
		}
		dimensions := DimensionSet{{Name: "x", Size: 4, TileSize: 4}}

		layer := NewLayer("test", true, CompressionNone, dimensions, fields) // separated mode

		wrtBuf := buffer.NewBuffer(100)
		iterator := NewTileOrderWriteIterator(wrtBuf, header, layer)

		testData := []struct {
			flag1 bool
			value int16
			flag2 bool
		}{
			{true, 100, false},
			{false, 200, true},
			{true, 300, true},
			{false, 400, false},
		}

		sampleIndex := 0
		for iterator.Next() {
			expectedData := testData[sampleIndex]

			// Use SetField() for each field
			iterator.SetField(0, expectedData.flag1)
			iterator.SetField(1, expectedData.value)
			iterator.SetField(2, expectedData.flag2)

			sampleIndex++
		}

		iterator.Done()

		if iterator.Error() != nil {
			t.Fatalf("Write iterator encountered error: %v", iterator.Error())
		}

		// Verify written data
		rdBuf := buffer.NewBufferFrom(wrtBuf.Bytes())
		memLayer := NewMemoryLayer(rdBuf, header, layer)

		for i, expectedData := range testData {
			coord := SampleCoordinate{i}

			flag1Val, err := memLayer.FieldAt(coord, 0)
			if err != nil {
				t.Fatalf("FieldAt(0) failed at coord %v: %v", coord, err)
			}
			valueVal, err := memLayer.FieldAt(coord, 1)
			if err != nil {
				t.Fatalf("FieldAt(1) failed at coord %v: %v", coord, err)
			}
			flag2Val, err := memLayer.FieldAt(coord, 2)
			if err != nil {
				t.Fatalf("FieldAt(2) failed at coord %v: %v", coord, err)
			}

			if flag1Val.(bool) != expectedData.flag1 {
				t.Errorf("Flag1 field mismatch at coord %v: expected %v, got %v", coord, expectedData.flag1, flag1Val)
			}
			if valueVal.(int16) != expectedData.value {
				t.Errorf("Value field mismatch at coord %v: expected %v, got %v", coord, expectedData.value, valueVal)
			}
			if flag2Val.(bool) != expectedData.flag2 {
				t.Errorf("Flag2 field mismatch at coord %v: expected %v, got %v", coord, expectedData.flag2, flag2Val)
			}
		}
	})
}
