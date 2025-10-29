package pixi

// Represents a linear index into the samples of a DimensionSet. This is the result of converting
// the multidimensional SampleCoordinate into a single integer in the range [0, Samples()).
type SampleIndex int

// Convert this linear SampleIndex into a multidimensional SampleCoordinate for the given DimensionSet.
func (index SampleIndex) ToSampleCoordinate(set DimensionSet) SampleCoordinate {
	coord := make([]int, len(set))
	total := set.Samples()
	for i := len(coord) - 1; i >= 0; i-- {
		total /= set[i].Size
		c := int(index) / total
		index %= SampleIndex(total)
		coord[i] = c
	}
	return coord
}

// Represents a multidimensional coordinate into the samples of a DimensionSet. Each element of the coordinate
// indexes into the corresponding dimension of the DimensionSet, and must be in the range [0, Size).
type SampleCoordinate []int

// Convert this multidimensional SampleCoordinate into an equivalent linear SampleIndex for the given DimensionSet.
func (coord SampleCoordinate) ToSampleIndex(set DimensionSet) SampleIndex {
	if len(coord) != len(set) {
		panic("pixi: SampleCoordinate dimension mismatch")
	}
	index := coord[len(coord)-1]
	for i := len(coord) - 2; i >= 0; i-- {
		index *= set[i].Size
		index += coord[i]
	}
	return SampleIndex(index)
}

// Convert this multidimensional SampleCoordinate into an equivalent TileSelector for the given DimensionSet.
func (coord SampleCoordinate) ToTileSelector(set DimensionSet) TileSelector {
	if len(coord) != len(set) {
		panic("pixi: SampleCoordinate dimension mismatch")
	}
	tileIndex := 0
	inTileIndex := 0
	tileMul := 1
	inTileMul := 1
	for dim := range coord {
		tileIndex += (coord[dim] / set[dim].TileSize) * tileMul
		inTileIndex += (coord[dim] % set[dim].TileSize) * inTileMul
		tileMul *= set[dim].Tiles()
		inTileMul *= set[dim].TileSize
	}
	return TileSelector{Tile: tileIndex, InTile: inTileIndex}
}

// Convert this multidimensional SampleCoordinate into an equivalent TileCoordinate for the given DimensionSet.
func (coord SampleCoordinate) ToTileCoordinate(set DimensionSet) TileCoordinate {
	if len(coord) != len(set) {
		panic("pixi: SampleCoordinate dimension mismatch")
	}
	tile := make([]int, len(set))
	inTile := make([]int, len(set))
	for i := range coord {
		tile[i] = coord[i] / set[i].TileSize
		inTile[i] = coord[i] % set[i].TileSize
	}
	return TileCoordinate{tile, inTile}

}

type TileOrderIndex int

// Indexes into a sample of a particular tile in the DimensionSet. The Tile field is a linear index into the tiles of the DimensionSet,
// and the InTile field is a linear index into the samples of that tile. The Tile field is converted from a multidimensional TileCoordinate
// in a similar way to SampleIndex being converted from SampleCoordinate, and the same for the InTile linear index. Note that Tile here is
// NOT guaranteed to be usable as a TileIndex for a Layer, since that requires knowledge of whether the Layer is separated or not.
type TileSelector struct {
	Tile   int
	InTile int
}

// Convert this TileSelector into a linear TileIndex for the given DimensionSet.
func (s TileSelector) ToTileIndex(set DimensionSet) TileOrderIndex {
	return TileOrderIndex(set.TileSamples()*s.Tile + s.InTile)
}

// Convert this TileSelector into a multidimensional TileCoordinate for the given DimensionSet.
func (s TileSelector) ToTileCoordinate(set DimensionSet) TileCoordinate {
	coord := TileCoordinate{make([]int, len(set)), make([]int, len(set))}
	tileIndex := s.Tile
	inTileIndex := s.InTile
	totalTiles := set.Tiles()
	totalSamples := set.TileSamples()
	for i := len(set) - 1; i >= 0; i-- {
		totalTiles /= set[i].Tiles()
		totalSamples /= set[i].TileSize
		tileCoord := tileIndex / totalTiles
		inTileCoord := inTileIndex / totalSamples
		tileIndex %= totalTiles
		inTileIndex %= totalSamples
		coord.Tile[i] = tileCoord
		coord.InTile[i] = inTileCoord
	}
	return coord
}

// Represents a multidimensional coordinate into the tiles of a DimensionSet, and a multidimensional coordinate into the samples
// of that tile. Each element of the Tile coordinate indexes into the corresponding dimension of the DimensionSet, and must be
// in the range [0, Tiles()). Each element of the InTile coordinate indexes into the corresponding dimension of the tile, and must be
// in the range [0, TileSize).
type TileCoordinate struct {
	Tile   []int
	InTile []int
}

// Convert this TileCoordinate into an equivalent TileSelector for the given DimensionSet.
func (coord TileCoordinate) ToTileSelector(set DimensionSet) TileSelector {
	tile := coord.Tile[len(coord.Tile)-1]
	inTile := coord.InTile[len(coord.InTile)-1]
	for i := len(coord.Tile) - 2; i >= 0; i-- {
		tile *= set[i].Tiles()
		tile += coord.Tile[i]
		inTile *= set[i].TileSize
		inTile += coord.InTile[i]
	}
	return TileSelector{Tile: tile, InTile: inTile}
}

// Convert this TileCoordinate into an equivalent SampleCoordinate for the given DimensionSet.
func (coord TileCoordinate) ToSampleCoordinate(set DimensionSet) SampleCoordinate {
	sampleCoord := make(SampleCoordinate, len(coord.Tile))
	for i := range sampleCoord {
		sampleCoord[i] = coord.Tile[i]*set[i].TileSize + coord.InTile[i]
	}
	return sampleCoord
}
