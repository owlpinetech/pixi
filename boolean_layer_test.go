package pixi

import (
	"encoding/binary"
	"testing"

	"github.com/owlpinetech/pixi/internal/buffer"
)

func TestMemoryLayerBooleanChannels(t *testing.T) {
	header := &Header{
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
			// Create layer with boolean and other channels
			channels := ChannelSet{
				{Name: "active", Type: ChannelBool},
				{Name: "count", Type: ChannelInt32},
				{Name: "enabled", Type: ChannelBool},
			}
			dimensions := DimensionSet{{Name: "x", Size: 10, TileSize: 5}}

			opts := []LayerOption{}
			if mode.separated {
				opts = append(opts, WithPlanar())
			}
			layer := NewLayer("test", dimensions, channels, opts...)

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
					err := SetSampleAt(memLayer, coord, sample)
					if err != nil {
						t.Fatalf("SetSampleAt failed at coord %v: %v", coord, err)
					}

					// Read it back
					readSample, err := SampleAt(memLayer, coord)
					if err != nil {
						t.Fatalf("SampleAt failed at coord %v: %v", coord, err)
					}

					// Verify each channel
					if readSample[0].(bool) != data.active {
						t.Errorf("Active channel mismatch at %v: expected %v, got %v", coord, data.active, readSample[0])
					}
					if readSample[1].(int32) != data.count {
						t.Errorf("Count channel mismatch at %v: expected %v, got %v", coord, data.count, readSample[1])
					}
					if readSample[2].(bool) != data.enabled {
						t.Errorf("Enabled channel mismatch at %v: expected %v, got %v", coord, data.enabled, readSample[2])
					}
				}
			})

			// Test ChannelAt and SetChannelAt
			t.Run("ChannelAt_SetChannelAt", func(t *testing.T) {
				for i, data := range testData {
					coord := SampleCoordinate{i}

					// Set individual channels
					err := SetChannelAt(memLayer, coord, 0, data.active)
					if err != nil {
						t.Fatalf("SetChannelAt(active) failed at coord %v: %v", coord, err)
					}
					err = SetChannelAt(memLayer, coord, 1, data.count)
					if err != nil {
						t.Fatalf("SetChannelAt(count) failed at coord %v: %v", coord, err)
					}
					err = SetChannelAt(memLayer, coord, 2, data.enabled)
					if err != nil {
						t.Fatalf("SetChannelAt(enabled) failed at coord %v: %v", coord, err)
					}

					// Read individual channels
					activeVal, err := ChannelAt(memLayer, coord, 0)
					if err != nil {
						t.Fatalf("ChannelAt(active) failed at coord %v: %v", coord, err)
					}
					countVal, err := ChannelAt(memLayer, coord, 1)
					if err != nil {
						t.Fatalf("ChannelAt(count) failed at coord %v: %v", coord, err)
					}
					enabledVal, err := ChannelAt(memLayer, coord, 2)
					if err != nil {
						t.Fatalf("ChannelAt(enabled) failed at coord %v: %v", coord, err)
					}

					// Verify values
					if activeVal.(bool) != data.active {
						t.Errorf("ChannelAt(active) mismatch at %v: expected %v, got %v", coord, data.active, activeVal)
					}
					if countVal.(int32) != data.count {
						t.Errorf("ChannelAt(count) mismatch at %v: expected %v, got %v", coord, data.count, countVal)
					}
					if enabledVal.(bool) != data.enabled {
						t.Errorf("ChannelAt(enabled) mismatch at %v: expected %v, got %v", coord, data.enabled, enabledVal)
					}
				}
			})
		})
	}
}

func TestCachedLayerBooleanChannels(t *testing.T) {
	header := &Header{
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
			// Create layer with boolean and other channels
			channels := ChannelSet{
				{Name: "visible", Type: ChannelBool},
				{Name: "priority", Type: ChannelInt16},
				{Name: "active", Type: ChannelBool},
			}
			dimensions := DimensionSet{{Name: "x", Size: 8, TileSize: 4}}

			opts := []LayerOption{}
			if mode.separated {
				opts = append(opts, WithPlanar())
			}
			layer := NewLayer("test", dimensions, channels, opts...)

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
			cache := NewFifoCacheLayer(rdBuf, header, layer, 100)

			// Test SampleAt and SetSampleAt
			t.Run("SampleAt_SetSampleAt", func(t *testing.T) {
				for i, data := range testData {
					coord := SampleCoordinate{i}
					sample := Sample{data.visible, data.priority, data.active}

					// Set the sample
					err := SetSampleAt(cache, coord, sample)
					if err != nil {
						t.Fatalf("SetSampleAt failed at coord %v: %v", coord, err)
					}

					// Read it back
					readSample, err := SampleAt(cache, coord)
					if err != nil {
						t.Fatalf("SampleAt failed at coord %v: %v", coord, err)
					}

					// Verify each channel
					if readSample[0].(bool) != data.visible {
						t.Errorf("Visible channel mismatch at %v: expected %v, got %v", coord, data.visible, readSample[0])
					}
					if readSample[1].(int16) != data.priority {
						t.Errorf("Priority channel mismatch at %v: expected %v, got %v", coord, data.priority, readSample[1])
					}
					if readSample[2].(bool) != data.active {
						t.Errorf("Active channel mismatch at %v: expected %v, got %v", coord, data.active, readSample[2])
					}
				}
			})

			// Test ChannelAt and SetChannelAt
			t.Run("ChannelAt_SetChannelAt", func(t *testing.T) {
				for i, data := range testData {
					coord := SampleCoordinate{i}

					// Set individual channels
					err := SetChannelAt(cache, coord, 0, data.visible)
					if err != nil {
						t.Fatalf("SetChannelAt(visible) failed at coord %v: %v", coord, err)
					}
					err = SetChannelAt(cache, coord, 1, data.priority)
					if err != nil {
						t.Fatalf("SetChannelAt(priority) failed at coord %v: %v", coord, err)
					}
					err = SetChannelAt(cache, coord, 2, data.active)
					if err != nil {
						t.Fatalf("SetChannelAt(active) failed at coord %v: %v", coord, err)
					}

					// Read individual channels
					visibleVal, err := ChannelAt(cache, coord, 0)
					if err != nil {
						t.Fatalf("ChannelAt(visible) failed at coord %v: %v", coord, err)
					}
					priorityVal, err := ChannelAt(cache, coord, 1)
					if err != nil {
						t.Fatalf("ChannelAt(priority) failed at coord %v: %v", coord, err)
					}
					activeVal, err := ChannelAt(cache, coord, 2)
					if err != nil {
						t.Fatalf("ChannelAt(active) failed at coord %v: %v", coord, err)
					}

					// Verify values
					if visibleVal.(bool) != data.visible {
						t.Errorf("ChannelAt(visible) mismatch at %v: expected %v, got %v", coord, data.visible, visibleVal)
					}
					if priorityVal.(int16) != data.priority {
						t.Errorf("ChannelAt(priority) mismatch at %v: expected %v, got %v", coord, data.priority, priorityVal)
					}
					if activeVal.(bool) != data.active {
						t.Errorf("ChannelAt(active) mismatch at %v: expected %v, got %v", coord, data.active, activeVal)
					}
				}
			})
		})
	}
}

func TestBooleanChannelBitPacking(t *testing.T) {
	header := &Header{
		Version:    Version,
		ByteOrder:  binary.BigEndian,
		OffsetSize: 8,
	}

	// Test specifically the bitchannel packing in separated mode
	channels := ChannelSet{
		{Name: "flags", Type: ChannelBool},
	}
	// Use 10 samples to test byte boundary (8 bits + 2 bits)
	dimensions := DimensionSet{{Name: "x", Size: 10, TileSize: 10}}

	layer := NewLayer("test", dimensions, channels, WithPlanar()) // separated = true

	// Verify tile size is correctly calculated for bitchannel
	expectedTileSize := (10 + 7) / 8 // 10 bits = 2 bytes
	actualTileSize := layer.DiskTileSize(0)
	if actualTileSize != expectedTileSize {
		t.Errorf("DiskTileSize for boolean bitchannel: expected %d bytes, got %d", expectedTileSize, actualTileSize)
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
		err := SetChannelAt(memLayer, coord, 0, value)
		if err != nil {
			t.Fatalf("SetChannelAt failed at coord %d: %v", i, err)
		}
	}

	// Read back and verify
	for i, expected := range testPattern {
		coord := SampleCoordinate{i}
		value, err := ChannelAt(memLayer, coord, 0)
		if err != nil {
			t.Fatalf("ChannelAt failed at coord %d: %v", i, err)
		}
		if value.(bool) != expected {
			t.Errorf("Bit pattern mismatch at position %d: expected %v, got %v", i, expected, value)
		}
	}
}

func TestTileOrderReadIteratorBooleanChannels(t *testing.T) {
	header := &Header{
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
			// Create layer with boolean and other channels
			channels := ChannelSet{
				{Name: "enabled", Type: ChannelBool},
				{Name: "score", Type: ChannelInt32},
				{Name: "active", Type: ChannelBool},
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
			opts := []LayerOption{}
			if mode.separated {
				opts = append(opts, WithPlanar())
			}
			layer, err := newBlankUncompressedLayer(
				wrtBuf,
				header,
				"tile-order-read-iterator-boolean-test",
				dimensions,
				channels,
				opts...,
			)
			if err != nil {
				t.Fatal(err)
			}

			// Create memory layer and set test data
			memLayer := NewMemoryLayer(wrtBuf, header, layer)

			for i, data := range testData {
				coord := SampleCoordinate{i}
				sample := Sample{data.enabled, data.score, data.active}
				err := SetSampleAt(memLayer, coord, sample)
				if err != nil {
					t.Fatalf("SetSampleAt failed at coord %v: %v", coord, err)
				}
			}
			memLayer.Commit()

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
					t.Errorf("Sample() enabled channel mismatch at coord %v: expected %v, got %v", coord, expectedData.enabled, sample[0])
				}
				if sample[1].(int32) != expectedData.score {
					t.Errorf("Sample() score channel mismatch at coord %v: expected %v, got %v", coord, expectedData.score, sample[1])
				}
				if sample[2].(bool) != expectedData.active {
					t.Errorf("Sample() active channel mismatch at coord %v: expected %v, got %v", coord, expectedData.active, sample[2])
				}

				// Test Channel() method
				enabledVal := iterator.Channel(0)
				scoreVal := iterator.Channel(1)
				activeVal := iterator.Channel(2)

				if enabledVal.(bool) != expectedData.enabled {
					t.Errorf("Channel(0) enabled channel mismatch at coord %v: expected %v, got %v", coord, expectedData.enabled, enabledVal)
				}
				if scoreVal.(int32) != expectedData.score {
					t.Errorf("Channel(1) score channel mismatch at coord %v: expected %v, got %v", coord, expectedData.score, scoreVal)
				}
				if activeVal.(bool) != expectedData.active {
					t.Errorf("Channel(2) active channel mismatch at coord %v: expected %v, got %v", coord, expectedData.active, activeVal)
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

func TestTileOrderWriteIteratorBooleanChannels(t *testing.T) {
	header := &Header{
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
			// Create layer with boolean and other channels
			channels := ChannelSet{
				{Name: "visible", Type: ChannelBool},
				{Name: "weight", Type: ChannelFloat32},
				{Name: "locked", Type: ChannelBool},
			}
			dimensions := DimensionSet{{Name: "x", Size: 8, TileSize: 4}}

			opts := []LayerOption{}
			if mode.separated {
				opts = append(opts, WithPlanar())
			}
			layer := NewLayer("test", dimensions, channels, opts...)

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
				sample, err := SampleAt(memLayer, coord)
				if err != nil {
					t.Fatalf("SampleAt failed at coord %v: %v", coord, err)
				}

				if sample[0].(bool) != expectedData.visible {
					t.Errorf("Visible channel mismatch at coord %v: expected %v, got %v", coord, expectedData.visible, sample[0])
				}
				if sample[1].(float32) != expectedData.weight {
					t.Errorf("Weight channel mismatch at coord %v: expected %v, got %v", coord, expectedData.weight, sample[1])
				}
				if sample[2].(bool) != expectedData.locked {
					t.Errorf("Locked channel mismatch at coord %v: expected %v, got %v", coord, expectedData.locked, sample[2])
				}
			}
		})
	}

	// Test SetChannel() method specifically
	t.Run("SetChannel", func(t *testing.T) {
		channels := ChannelSet{
			{Name: "flag1", Type: ChannelBool},
			{Name: "value", Type: ChannelInt16},
			{Name: "flag2", Type: ChannelBool},
		}
		dimensions := DimensionSet{{Name: "x", Size: 4, TileSize: 4}}

		layer := NewLayer("test", dimensions, channels, WithPlanar()) // separated mode

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

			// Use SetChannel() for each channel
			iterator.SetChannel(0, expectedData.flag1)
			iterator.SetChannel(1, expectedData.value)
			iterator.SetChannel(2, expectedData.flag2)

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

			flag1Val, err := ChannelAt(memLayer, coord, 0)
			if err != nil {
				t.Fatalf("ChannelAt(0) failed at coord %v: %v", coord, err)
			}
			valueVal, err := ChannelAt(memLayer, coord, 1)
			if err != nil {
				t.Fatalf("ChannelAt(1) failed at coord %v: %v", coord, err)
			}
			flag2Val, err := ChannelAt(memLayer, coord, 2)
			if err != nil {
				t.Fatalf("ChannelAt(2) failed at coord %v: %v", coord, err)
			}

			if flag1Val.(bool) != expectedData.flag1 {
				t.Errorf("Flag1 channel mismatch at coord %v: expected %v, got %v", coord, expectedData.flag1, flag1Val)
			}
			if valueVal.(int16) != expectedData.value {
				t.Errorf("Value channel mismatch at coord %v: expected %v, got %v", coord, expectedData.value, valueVal)
			}
			if flag2Val.(bool) != expectedData.flag2 {
				t.Errorf("Flag2 channel mismatch at coord %v: expected %v, got %v", coord, expectedData.flag2, flag2Val)
			}
		}
	})
}
