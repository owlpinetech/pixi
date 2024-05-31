package pixi

import "testing"

func TestMemoryDimIndicesToTileIndices(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memSet := &InMemoryDataset{}
			memSet.Dimensions = tt.dimensions
			tileIndex, inTileIndex := memSet.dimIndicesToTileIndices(tt.dimIndices)
			if tileIndex != tt.expectedTileIndex || inTileIndex != tt.expectedInTileIndex {
				t.Errorf("dimIndicesToTileIndices() = (%d, %d), want (%d, %d)", tileIndex, inTileIndex, tt.expectedTileIndex, tt.expectedInTileIndex)
			}
		})
	}
}
