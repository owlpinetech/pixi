package gopixi

import (
	"math/rand"
	"testing"
)

func TestDimensionSetIndicesSampleOrder(t *testing.T) {
	dimCount := rand.Intn(5)
	dims := make(DimensionSet, dimCount)
	for i := range dims {
		size := rand.Intn(99) + 1
		tileSize := size / (rand.Intn(5) + 1)
		if tileSize == 0 {
			tileSize = size
		}
		dims[i] = Dimension{Size: size, TileSize: tileSize}
	}

	sampleInd := SampleIndex(0)
	for coord := range dims.SampleCoordinates() {
		if coord.ToSampleIndex(dims) != sampleInd {
			t.Fatalf("expected %v to be sample index %d, but got %d", coord, sampleInd, coord.ToSampleIndex(dims))
		}
		sampleInd++
	}
}

func TestDimensionSetIndicesTileOrder(t *testing.T) {
	dims := DimensionSet{{"", 15, 5}, {"", 60, 30}} //newRandomValidDimensionSet(5, 99, 5)

	tileInd := TileOrderIndex(0)
	for coord := range dims.TileCoordinates() {
		if coord.ToTileSelector(dims).ToTileIndex(dims) != tileInd {
			t.Fatalf("expected %v to be sample index %d, but got %d for %v", coord, tileInd, coord.ToTileSelector(dims).ToTileIndex(dims), dims)
		}
		tileInd++
	}
}

func TestDimensionSetContainsCoordinate(t *testing.T) {
	dims := DimensionSet{{"", 10, 5}, {"", 20, 10}}

	tests := []struct {
		coord    SampleCoordinate
		expected bool
	}{
		{SampleCoordinate{0, 0}, true},
		{SampleCoordinate{5, 10}, true},
		{SampleCoordinate{9, 19}, true},
		{SampleCoordinate{10, 0}, false},
		{SampleCoordinate{0, 20}, false},
		{SampleCoordinate{-1, 0}, false},
		{SampleCoordinate{0, -1}, false},
		{SampleCoordinate{5}, false},
		{SampleCoordinate{5, 10, 15}, false},
	}

	for _, test := range tests {
		result := dims.ContainsCoordinate(test.coord)
		if result != test.expected {
			t.Errorf("ContainsCoordinate(%v) = %v; expected %v", test.coord, result, test.expected)
		}
	}
}
