package pixi

import (
	"math/rand"
	"slices"
	"testing"
)

func TestDimensionCoordinateToSampleIndex(t *testing.T) {
	tests := []struct {
		dims   DimensionSet
		coords []SampleCoordinate
		expect []SampleIndex
	}{
		{
			dims:   []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
			coords: []SampleCoordinate{{0, 0}, {3, 0}, {0, 3}, {3, 3}},
			expect: []SampleIndex{0, 3, 12, 15},
		},
		{
			dims: []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
			coords: []SampleCoordinate{
				{0, 0, 0}, {3, 0, 0}, {0, 3, 0}, {0, 0, 3},
				{3, 3, 0}, {3, 0, 3}, {0, 3, 3}, {3, 3, 3},
			},
			expect: []SampleIndex{0, 3, 12, 48, 15, 51, 60, 63},
		},
	}

	for _, tc := range tests {
		for i, c := range tc.coords {
			got := c.ToSampleIndex(tc.dims)
			if got != tc.expect[i] {
				t.Errorf("expected index %d for coord %v, got %d", tc.expect[i], c, got)
			}
		}
	}
}

func TestSampleIndexToDimensionCoordinate(t *testing.T) {
	tests := []struct {
		dims   DimensionSet
		index  []SampleIndex
		expect []SampleCoordinate
	}{
		{
			dims:   []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
			expect: []SampleCoordinate{{0, 0}, {3, 0}, {0, 3}, {3, 3}},
			index:  []SampleIndex{0, 3, 12, 15},
		},
		{
			dims: []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
			expect: []SampleCoordinate{
				{0, 0, 0}, {3, 0, 0}, {0, 3, 0}, {0, 0, 3},
				{3, 3, 0}, {3, 0, 3}, {0, 3, 3}, {3, 3, 3},
			},
			index: []SampleIndex{0, 3, 12, 48, 15, 51, 60, 63},
		},
	}

	for _, tc := range tests {
		for i, ind := range tc.index {
			got := ind.ToSampleCoordinate(tc.dims)
			if !slices.Equal(got, tc.expect[i]) {
				t.Errorf("expected coord %v for index %d, got %v", tc.expect[i], ind, got)
			}
		}
	}
}

func TestSampleCoordinateToTileCoordinateAndBack(t *testing.T) {
	dims := newRandomValidDimensionSet(5, 99, 5)

	for range 50 {
		sampleCoord := make(SampleCoordinate, len(dims))
		for i := range sampleCoord {
			sampleCoord[i] = rand.Intn(dims[i].Size)
		}

		tileCoord := sampleCoord.ToTileCoordinate(dims)
		resSample := tileCoord.ToSampleCoordinate(dims)
		if !slices.Equal(sampleCoord, resSample) {
			t.Fatalf("expected same coord %v, but got %v for dims %v", sampleCoord, resSample, dims)
		}
	}
}

func TestTileCoordinateToTileSelectorAndBack(t *testing.T) {
	dims := newRandomValidDimensionSet(5, 99, 5)

	for range 50 {
		tileCoord := TileCoordinate{make([]int, len(dims)), make([]int, len(dims))}
		for i := range tileCoord.Tile {
			tileCoord.Tile[i] = rand.Intn(dims[i].Tiles())
			tileCoord.InTile[i] = rand.Intn(dims[i].TileSize)
		}

		tileSelect := tileCoord.ToTileSelector(dims)
		resCoord := tileSelect.ToTileCoordinate(dims)
		if !slices.Equal(tileCoord.Tile, resCoord.Tile) || !slices.Equal(tileCoord.InTile, resCoord.InTile) {
			t.Fatalf("expected same coord %v, but got %v for dims %v", tileSelect, resCoord, dims)
		}
	}
}

func TestSampleCoordinateToTileSelectorDirectAndIndirectSame(t *testing.T) {
	dims := newRandomValidDimensionSet(5, 99, 5)

	for range 50 {
		sampleCoord := make(SampleCoordinate, len(dims))
		for i := range sampleCoord {
			sampleCoord[i] = rand.Intn(dims[i].Size)
		}

		tileSelectDirect := sampleCoord.ToTileSelector(dims)
		tileSelectIndirect := sampleCoord.ToTileCoordinate(dims).ToTileSelector(dims)
		if tileSelectDirect != tileSelectIndirect {
			t.Fatalf("expected same tile selector direct %v and indirect %v for coord %v, dims %v", tileSelectDirect, tileSelectIndirect, sampleCoord, dims)
		}
	}
}
