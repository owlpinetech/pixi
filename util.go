package pixi

import "math/rand"

// Used in test packages to help generate random dimension sets for better test coverage.
func newRandomValidDimensionSet(maxDims int, maxDimSize int, maxTileSize int) DimensionSet {
	dimCount := rand.Intn(maxDims-1) + 1
	dims := make(DimensionSet, dimCount)
	for i := range dims {
		size := rand.Intn(maxDimSize) + 1
		tileSize := size / (rand.Intn(maxTileSize) + 1)
		if tileSize == 0 {
			tileSize = size
		}
		dims[i] = &Dimension{Size: size, TileSize: tileSize}
	}
	return DimensionSet(dims)
}
