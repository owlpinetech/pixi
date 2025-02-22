package pixi

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

	tileInd := TileIndex(0)
	for coord := range dims.TileCoordinates() {
		if coord.ToTileSelector(dims).ToTileIndex(dims) != tileInd {
			t.Fatalf("expected %v to be sample index %d, but got %d for %v", coord, tileInd, coord.ToTileSelector(dims).ToTileIndex(dims), dims)
		}
		tileInd++
	}
}
