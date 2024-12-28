package pixi

type Dimension struct {
	Name     string // Friendly name to refer to the dimension in the layer.
	Size     int    // The total number of elements in the dimension.
	TileSize int    // The size of the tiles in the dimension. Does not need to be a factor of Size.
}

// Returns the number of tiles in this dimension.
// The number of tiles is calculated by dividing the size of the dimension by the tile size,
// and then rounding up to the nearest whole number if there are any remaining bytes that do not fit into a full tile.
func (d Dimension) Tiles() int {
	tiles := d.Size / d.TileSize
	if d.Size%d.TileSize != 0 {
		tiles += 1
	}
	return tiles
}
