package pixi

type SampleIndex int

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

type SampleCoordinate []int

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

type TileIndex int

type TileSelector struct {
	Tile   int
	InTile int
}

func (s TileSelector) ToTileIndex(set DimensionSet) TileIndex {
	return TileIndex(set.TileSamples()*s.Tile + s.InTile)
}

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

type TileCoordinate struct {
	Tile   []int
	InTile []int
}

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

func (coord TileCoordinate) ToSampleCoordinate(set DimensionSet) SampleCoordinate {
	sampleCoord := make(SampleCoordinate, len(coord.Tile))
	for i := range sampleCoord {
		sampleCoord[i] = coord.Tile[i]*set[i].TileSize + coord.InTile[i]
	}
	return sampleCoord
}
