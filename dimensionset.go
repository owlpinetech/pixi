package pixi

import "iter"

// An ordered set of named dimensions present in the layer of a Pixi file.
type DimensionSet []Dimension

// Computes the number of non-separated tiles in the data set. This number is the same regardless
// of how the tiles are laid out on disk; use the DiskTiles() method to determine the number of
// tiles actually stored on disk. Note that DiskTiles() >= Tiles() by definition.
func (d DimensionSet) Tiles() int {
	tiles := 1
	for _, t := range d {
		tiles *= t.Tiles()
	}
	return tiles
}

// The number of samples per tile in the data set. Each tile has the same number of samples,
// regardless of if the data is stored separated or continguous.
func (d DimensionSet) TileSamples() int {
	if len(d) <= 0 {
		return 0
	}
	samples := 1
	for _, d := range d {
		samples *= d.TileSize
	}
	return samples
}

// The total number of samples in the data set. If the tile size of any dimension is not
// a multiple of the dimension size, the 'padding' samples are not included in the count.
func (d DimensionSet) Samples() int {
	if len(d) <= 0 {
		return 0
	}
	samples := 1
	for _, dim := range d {
		samples *= dim.Size
	}
	return samples
}

// Iterate over the sample indices of the dimensions in the order the dimensions are laid out. That is,
// the index increments all the way through the first dimension, then increments the second (nesting the first each time), then
// the third (nesting the second (nesting the first)), and so on. The first dimension changes the most frequently, the last
// dimension changes the least frequently.
func (set DimensionSet) SampleCoordinates() iter.Seq[SampleCoordinate] {
	return func(yield func(index SampleCoordinate) bool) {
		samples := set.Samples()
		curInd := make(SampleCoordinate, len(set))
		for range samples {
			if !yield(curInd) {
				break
			}
			for dInd := range curInd {
				// increment the lowest dimension
				curInd[dInd] += 1
				if curInd[dInd] >= set[dInd].Size {
					// carry over into the next dimension
					curInd[dInd] = 0
				} else {
					break
				}
			}
		}
	}
}

// Iterate over the indices of the dimensions in the 'tile order', such that all indices for the first tile are
// yielded before the second tile, and so on. While iterating within the tile, the first dimension changes the
// most frequently, all the way to the last which changes the least frequently. See the documentation for
// SampleCoordinates for more details on iteration within tile coordinates.
func (set DimensionSet) TileCoordinates() iter.Seq[TileCoordinate] {
	return func(yield func(index TileCoordinate) bool) {
		samples := set.Tiles() * set.TileSamples()
		curInd := TileCoordinate{make([]int, len(set)), make([]int, len(set))}
		for range samples {
			if !yield(curInd) {
				break
			}
			// increment in-tile coordinates
			needNextTile := true
			for dInd := range set {
				// increment the lowest dimension in the tile
				curInd.InTile[dInd] += 1
				if curInd.InTile[dInd] >= set[dInd].TileSize {
					// carry over into the next dimension
					curInd.InTile[dInd] = 0
				} else {
					needNextTile = false
					break
				}
			}
			// made it out of the in-tile loop, so we must need to increment overall tile coordinates
			if needNextTile {
				for dInd := range set {
					curInd.Tile[dInd] += 1
					if curInd.Tile[dInd] >= set[dInd].Tiles() {
						// carry over into the next dimension
						curInd.Tile[dInd] = 0
					} else {
						break
					}
				}
			}
		}
	}
}

func (set DimensionSet) ContainsCoordinate(coord SampleCoordinate) bool {
	if len(coord) != len(set) {
		return false
	}
	for i, dimCoord := range coord {
		if dimCoord < 0 || dimCoord >= set[i].Size {
			return false
		}
	}
	return true
}

func (set DimensionSet) String() string {
	str := "DimensionSet{"
	for i, dim := range set {
		if i > 0 {
			str += ", "
		}
		str += dim.String()
	}
	str += "}"
	return str
}
