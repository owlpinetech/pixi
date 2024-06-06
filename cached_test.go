package pixi

import "testing"

func TestCachedDimIndicesToTileIndices(t *testing.T) {
	tests := []struct {
		name                string
		dimensions          []Dimension
		dimIndices          []uint
		expectedTileIndex   uint
		expectedInTileIndex uint
	}{
		{
			name:                "simple case",
			dimensions:          []Dimension{{Size: 2, TileSize: 1}, {Size: 3, TileSize: 2}},
			dimIndices:          []uint{0, 0},
			expectedTileIndex:   0,
			expectedInTileIndex: 0,
		},
		{
			name:                "tile increment case",
			dimensions:          []Dimension{{Size: 8, TileSize: 2}, {Size: 6, TileSize: 2}},
			dimIndices:          []uint{2, 1},
			expectedTileIndex:   1,
			expectedInTileIndex: 2,
		},
		{
			name:                "furthest corner case",
			dimensions:          []Dimension{{Size: 8, TileSize: 2}, {Size: 6, TileSize: 2}},
			dimIndices:          []uint{7, 5},
			expectedTileIndex:   11,
			expectedInTileIndex: 3,
		},
		{
			name:                "furthest corner three dimensions",
			dimensions:          []Dimension{{Size: 8, TileSize: 2}, {Size: 6, TileSize: 2}, {Size: 4, TileSize: 2}},
			dimIndices:          []uint{7, 5, 3},
			expectedTileIndex:   23,
			expectedInTileIndex: 7,
		},
		{
			name:                "edge case",
			dimensions:          []Dimension{{Size: 7, TileSize: 3}, {Size: 8, TileSize: 4}},
			dimIndices:          []uint{5, 6},
			expectedTileIndex:   4,
			expectedInTileIndex: 8,
		},
		{
			name:                "gebco",
			dimensions:          []Dimension{{Size: 86400, TileSize: 21600 / 4}, {Size: 43200, TileSize: 21600 / 4}},
			dimIndices:          []uint{0, 5400},
			expectedTileIndex:   16,
			expectedInTileIndex: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memSet := &CacheDataset{}
			memSet.Dimensions = tt.dimensions
			tileIndex, inTileIndex := memSet.dimIndicesToTileIndices(tt.dimIndices)
			if tileIndex != tt.expectedTileIndex || inTileIndex != tt.expectedInTileIndex {
				t.Errorf("dimIndicesToTileIndices() = (%d, %d), want (%d, %d)", tileIndex, inTileIndex, tt.expectedTileIndex, tt.expectedInTileIndex)
			}
		})
	}
}

func TestCacheDatasetEvict(t *testing.T) {
	tileCache := make(map[uint]*CacheTile)
	appendDataset := &CacheDataset{TileCache: tileCache}

	// Test that evict returns nil when cache is empty
	if err := appendDataset.evict(); err != nil {
		t.Errorf("expected evict to return nil, got %v", err)
	}

	// Test that evict removes the first element from the cache
	appendDataset.TileCache = map[uint]*CacheTile{1: {}, 2: {}, 3: {}}
	if err := appendDataset.evict(); err != nil {
		t.Errorf("expected evict to return nil, got %v", err)
	}
	if len(appendDataset.TileCache) != 2 {
		t.Errorf("expected ReadCache to have length 2 after evict, has length %d", len(appendDataset.TileCache))
	}
	if err := appendDataset.evict(); err != nil {
		t.Errorf("expected evict to return nil, got %v", err)
	}
	if len(appendDataset.TileCache) != 1 {
		t.Errorf("expected ReadCache to have length 1 after second evict, has length %d", len(appendDataset.TileCache))
	}
}

func TestCacheAllReadAllSample(t *testing.T) {
	buf := NewBuffer(10)
	under := DataSet{
		Separated:   false,
		Compression: CompressionNone,
		Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
		Fields:      []Field{{Type: FieldFloat64}, {Type: FieldInt16}, {Type: FieldUint64}},
	}
	dataset, err := NewCacheDataset(under, buf, 2, 0)
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
			if len(dataset.TileCache) > int(dataset.MaxInCache) {
				t.Errorf("expected read cache length to be less than %d, got %d", dataset.MaxInCache, len(dataset.TileCache))
			}
		}
	}
}

func TestCacheAllReadAllSampleField(t *testing.T) {
	buf := NewBuffer(10)
	under := DataSet{
		Separated:   false,
		Compression: CompressionNone,
		Dimensions:  []Dimension{{Size: 8, TileSize: 2}, {Size: 8, TileSize: 2}},
		Fields:      []Field{{Type: FieldFloat64}, {Type: FieldInt16}, {Type: FieldUint64}},
	}
	dataset, err := NewCacheDataset(under, buf, 2, 0)
	if err != nil {
		t.Fatal(err)
	}

	ytiles := dataset.Dimensions[1].Tiles()
	xtiles := dataset.Dimensions[0].Tiles()
	for ytile := 0; ytile < ytiles; ytile++ {
		for xtile := 0; xtile < xtiles; xtile++ {
			for x := 0; x < int(dataset.Dimensions[0].TileSize); x++ {
				for y := 0; y < int(dataset.Dimensions[1].TileSize); y++ {
					xDimInd := uint(xtile*int(dataset.Dimensions[0].TileSize) + x)
					yDimInd := uint(ytile*int(dataset.Dimensions[1].TileSize) + y)
					err := dataset.SetSampleField([]uint{xDimInd, yDimInd}, 0, 1.2)
					if err != nil {
						t.Fatal(err)
					}
					err = dataset.SetSampleField([]uint{xDimInd, yDimInd}, 1, int16(-13))
					if err != nil {
						t.Fatal(err)
					}
					err = dataset.SetSampleField([]uint{xDimInd, yDimInd}, 2, uint64(54321))
					if err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}

	for x := 0; x < int(dataset.Dimensions[0].Size); x++ {
		for y := 0; y < int(dataset.Dimensions[1].Size); y++ {
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
			if len(dataset.TileCache) > int(dataset.MaxInCache) {
				t.Errorf("expected read cache length to be less than %d, got %d", dataset.MaxInCache, len(dataset.TileCache))
			}
		}
	}
}
