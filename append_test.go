package pixi

import (
	"reflect"
	"testing"
)

func TestWriteCompressTile(t *testing.T) {
	tests := []struct {
		name        string
		compression Compression
		data        []byte
		expected    []byte
	}{
		{
			name:        "small data with gzip compression",
			compression: CompressionGzip,
			data:        []byte{1, 2, 3, 4, 5},
			expected:    []byte{98, 100, 98, 102, 97, 5, 4, 0, 0, 255, 255},
		},
		{
			name:        "large data with none compression",
			compression: CompressionNone,
			data:        []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			expected:    []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := NewBuffer(10)
			under := DataSet{
				Separated:   false,
				Compression: tc.compression,
				Dimensions:  []Dimension{{Size: int64(len(tc.data)), TileSize: int32(len(tc.data))}},
				Fields:      []Field{{Name: "byte", Type: FieldUint8}},
			}
			dataset, err := NewAppendDataset(under, buf, 10, 0)
			if err != nil {
				t.Fatal(err)
			}
			err = dataset.writeCompressTile(tc.data, 0)
			if err != nil {
				t.Errorf("expected no error, but got %s", err)
			}

			if !reflect.DeepEqual(tc.expected, buf.Bytes()) {
				t.Errorf("expected written to be %v, but got %v", tc.expected, buf.Bytes())
			}
			if !reflect.DeepEqual(int(dataset.TileBytes[0]), len(tc.expected)) {
				t.Errorf("expected TileBytes to be %d, but got %d", len(tc.expected), dataset.TileBytes[0])
			}
		})
	}
}

func TestAppendDatasetEvict(t *testing.T) {
	readCache := make(map[uint]*AppendTile)
	appendDataset := &AppendDataset{ReadCache: readCache}

	// Test that evict returns nil when cache is empty
	if err := appendDataset.evict(); err != nil {
		t.Errorf("expected evict to return nil, got %v", err)
	}

	// Test that evict removes the first element from the cache
	appendDataset.ReadCache = map[uint]*AppendTile{1: {}, 2: {}, 3: {}}
	if err := appendDataset.evict(); err != nil {
		t.Errorf("expected evict to return nil, got %v", err)
	}
	if len(appendDataset.ReadCache) != 2 {
		t.Errorf("expected ReadCache to have length 2 after evict, has length %d", len(appendDataset.ReadCache))
	}
	if err := appendDataset.evict(); err != nil {
		t.Errorf("expected evict to return nil, got %v", err)
	}
	if len(appendDataset.ReadCache) != 1 {
		t.Errorf("expected ReadCache to have length 1 after second evict, has length %d", len(appendDataset.ReadCache))
	}
}

func TestAppendAddTileToCache(t *testing.T) {
	// Create a new AppendDataset with maxInCache set to 3
	d := &AppendDataset{MaxInCache: 3, ReadCache: make(map[uint]*AppendTile)}

	// Add tiles to the cache and verify they are added correctly
	for i := uint(0); i < 5; i++ {
		tileIndex := i
		tile := AppendTile{Data: []byte{byte(i)}}
		err := d.addTileToCache(tileIndex, tile)
		if err != nil {
			t.Errorf("addTileToCache failed: %v", err)
			return
		}
	}

	// Verify that the cache has 3 tiles (including eviction of older tiles if needed)
	if len(d.ReadCache) != 3 {
		t.Errorf("Expected ReadCache length to be 3, got %d", len(d.ReadCache))
		return
	}

	// Add another tile and verify it replaces the oldest tile in the cache
	for i := uint(5); i < 8; i++ {
		tileIndex := i
		tile := AppendTile{Data: []byte{byte(i)}}
		err := d.addTileToCache(tileIndex, tile)
		if err != nil {
			t.Errorf("addTileToCache failed: %v", err)
			return
		}
	}

	// Verify that the cache has 3 tiles (after evicting older tiles if needed)
	if len(d.ReadCache) != 3 {
		t.Errorf("Expected ReadCache length to be 3, got %d", len(d.ReadCache))
		return
	}
}

func TestAppendSetGetSeparatedSampleField(t *testing.T) {
	buf := NewBuffer(10)
	under := DataSet{
		Separated:   false,
		Compression: CompressionNone,
		Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
		Fields:      []Field{{Type: FieldFloat64}, {Type: FieldInt16}, {Type: FieldUint64}},
	}
	dataset, err := NewAppendDataset(under, buf, 10, 0)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name               string
		indices            []uint
		field              uint
		val                any
		expectedWriteIndex uint
	}{
		{
			name:               "first index, first field",
			indices:            []uint{0, 0},
			field:              0,
			val:                4.5,
			expectedWriteIndex: 0,
		},
		{
			name:               "first index, second field",
			indices:            []uint{0, 0},
			field:              1,
			val:                int16(-13),
			expectedWriteIndex: 0,
		},
		{
			name:               "first index, third field",
			indices:            []uint{0, 0},
			field:              2,
			val:                uint64(987654321),
			expectedWriteIndex: 0,
		},
		{
			name:               "third index, first field",
			indices:            []uint{0, 1},
			field:              0,
			val:                156.234,
			expectedWriteIndex: 0,
		},
		{
			name:               "second index, second field",
			indices:            []uint{1, 0},
			field:              1,
			val:                int16(-97),
			expectedWriteIndex: 0,
		},
		{
			name:               "fifth index, second field",
			indices:            []uint{3, 0},
			field:              1,
			val:                int16(1013),
			expectedWriteIndex: 1,
		},
	}

	for _, tc := range tests {
		err = dataset.SetSampleField(tc.indices, tc.field, tc.val)
		if err != nil {
			t.Fatalf("expected no error for SetSampleField, but got %s", err)
		}

		getVal, err := dataset.GetSampleField(tc.indices, tc.field)
		if err != nil {
			t.Fatalf("expected no error for GetSampleField, but got %s", err)
		}

		if getVal != tc.val {
			t.Errorf("expected to get %v after setting, got %v", tc.val, getVal)
		}
		if dataset.WritingTileIndex != tc.expectedWriteIndex {
			t.Errorf("expected writing tile index to be %d, got %d", tc.expectedWriteIndex, dataset.WritingTileIndex)
		}
	}
}

func TestAppendSetGetContinguousSample(t *testing.T) {
	buf := NewBuffer(10)
	under := DataSet{
		Separated:   false,
		Compression: CompressionNone,
		Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
		Fields:      []Field{{Type: FieldFloat64}, {Type: FieldInt16}, {Type: FieldUint64}},
	}
	dataset, err := NewAppendDataset(under, buf, 10, 0)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name               string
		indices            []uint
		sample             []any
		expectedWriteIndex uint
	}{
		{
			name:               "first index",
			indices:            []uint{0, 0},
			sample:             []any{4.5, int16(-13), uint64(987654321)},
			expectedWriteIndex: 0,
		},
		{
			name:               "third index",
			indices:            []uint{0, 1},
			sample:             []any{156.234, int16(78), uint64(0)},
			expectedWriteIndex: 0,
		},
		{
			name:               "second index",
			indices:            []uint{1, 0},
			sample:             []any{0.0001, int16(-97), uint64(11111)},
			expectedWriteIndex: 0,
		},
		{
			name:               "fifth index",
			indices:            []uint{3, 0},
			sample:             []any{18.03, int16(1013), uint64(02)},
			expectedWriteIndex: 1,
		},
	}

	for _, tc := range tests {
		err = dataset.SetSample(tc.indices, tc.sample)
		if err != nil {
			t.Fatalf("expected no error for SetSampleField, but got %s", err)
		}

		getVal, err := dataset.GetSample(tc.indices)
		if err != nil {
			t.Fatalf("expected no error for GetSampleField, but got %s", err)
		}

		if !reflect.DeepEqual(tc.sample, getVal) {
			t.Errorf("expected written to be %v, but got %v", tc.sample, getVal)
		}
		if dataset.WritingTileIndex != tc.expectedWriteIndex {
			t.Errorf("expected writing tile index to be %d, got %d", tc.expectedWriteIndex, dataset.WritingTileIndex)
		}
	}
}

func TestAppendAllReadAllSample(t *testing.T) {
	buf := NewBuffer(10)
	under := DataSet{
		Separated:   false,
		Compression: CompressionGzip,
		Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
		Fields:      []Field{{Type: FieldFloat64}, {Type: FieldInt16}, {Type: FieldUint64}},
	}
	dataset, err := NewAppendDataset(under, buf, 1, 0)
	if err != nil {
		t.Fatal(err)
	}

	for ytile := 0; ytile < 2; ytile++ {
		for xtile := 0; xtile < 2; xtile++ {
			for x := 0; x < 2; x++ {
				for y := 0; y < 2; y++ {
					err := dataset.SetSample([]uint{uint(xtile*2 + x), uint(ytile*2 + y)}, []any{1.2, int16(-13), uint64(54321)})
					if err != nil {
						t.Fatal(err)
					}
					if dataset.WritingTileIndex != uint(xtile)+uint(ytile)*2 {
						t.Errorf("expected %d,%d tile index to be %d, got %d", xtile, ytile, uint(xtile)+uint(ytile)*2, dataset.WritingTileIndex)
					}
				}
			}
		}
	}

	for x := 0; x < 4; x++ {
		for y := 0; y < 4; y++ {
			val, err := dataset.GetSample([]uint{uint(x), uint(y)})
			if err != nil {
				t.Fatalf("failed to get sample: %s", err)
			}
			if val[0].(float64) != 1.2 {
				t.Errorf("expected first sample field to be 1.2, got %v", val[0])
			}
			if val[1].(int16) != int16(-13) {
				t.Errorf("expected second sample field to be -13, got %v", val[1])
			}
			if val[2].(uint64) != uint64(54321) {
				t.Errorf("expected third sample field to be 54321, got %v", val[2])
			}
			if len(dataset.ReadCache) > int(dataset.MaxInCache) {
				t.Errorf("expected read cache length to be less than %d, got %d", dataset.MaxInCache, len(dataset.ReadCache))
			}
		}
	}
}

func TestAppendllReadAllSampleField(t *testing.T) {
	buf := NewBuffer(10)
	under := DataSet{
		Separated:   false,
		Compression: CompressionNone,
		Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
		Fields:      []Field{{Type: FieldFloat64}, {Type: FieldInt16}, {Type: FieldUint64}},
	}
	dataset, err := NewAppendDataset(under, buf, 2, 0)
	if err != nil {
		t.Fatal(err)
	}

	for ytile := 0; ytile < 2; ytile++ {
		for xtile := 0; xtile < 2; xtile++ {
			for x := 0; x < 2; x++ {
				for y := 0; y < 2; y++ {
					err := dataset.SetSampleField([]uint{uint(xtile*2 + x), uint(ytile*2 + y)}, 0, 1.2)
					if err != nil {
						t.Fatal(err)
					}
					err = dataset.SetSampleField([]uint{uint(xtile*2 + x), uint(ytile*2 + y)}, 1, int16(-13))
					if err != nil {
						t.Fatal(err)
					}
					err = dataset.SetSampleField([]uint{uint(xtile*2 + x), uint(ytile*2 + y)}, 2, uint64(54321))
					if err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}

	for x := 0; x < 4; x++ {
		for y := 0; y < 4; y++ {
			val0, err := dataset.GetSampleField([]uint{uint(x), uint(y)}, 0)
			if err != nil {
				t.Fatalf("failed to get sample 0: %s", err)
			}
			val1, err := dataset.GetSampleField([]uint{uint(x), uint(y)}, 1)
			if err != nil {
				t.Fatalf("failed to get sample 1: %s", err)
			}
			val2, err := dataset.GetSampleField([]uint{uint(x), uint(y)}, 2)
			if err != nil {
				t.Fatalf("failed to get sample 2: %s", err)
			}
			if val0.(float64) != 1.2 {
				t.Errorf("expected first sample field to be 1.2, got %v", val0)
			}
			if val1.(int16) != int16(-13) {
				t.Errorf("expected second sample field to be -13, got %v", val1)
			}
			if val2.(uint64) != uint64(54321) {
				t.Errorf("expected third sample field to be 54321, got %v", val2)
			}
			if len(dataset.ReadCache) > int(dataset.MaxInCache) {
				t.Errorf("expected read cache length to be less than %d, got %d", dataset.MaxInCache, len(dataset.ReadCache))
			}
		}
	}
}
