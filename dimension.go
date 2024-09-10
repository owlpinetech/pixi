package pixi

type Dimension struct {
	Size     int
	TileSize int
}

// Returns the number of tiles in this dimension.
// The number of tiles is calculated by dividing the size of the dimension by the tile size,
// and then rounding up to the nearest whole number if there are any remaining bytes that do not fit into a full tile.
func (d Dimension) Tiles() int {
	if d.Size <= 0 {
		return 0
	}
	if d.TileSize <= 0 {
		panic("pixi: Size of dimension > 0 but TileSize set to 0, invalid")
	}
	tiles := d.Size / d.TileSize
	if d.Size%d.TileSize != 0 {
		tiles += 1
	}
	return tiles
}
