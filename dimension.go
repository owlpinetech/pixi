package pixi

import (
	"io"
	"iter"
)

// Represents an axis along which tiled, gridded data is sstored in a Pixi file. Data sets can have
// one or more dimensions, but never zero. If a dimension is not tiled, then the TileSize should be
// the same as a the total Size.
type Dimension struct {
	Name     string // Friendly name to refer to the dimension in the layer.
	Size     int    // The total number of elements in the dimension.
	TileSize int    // The size of the tiles in the dimension. Does not need to be a factor of Size.
}

// Get the size in bytes of this dimension description as it is laid out and written to disk.
func (d Dimension) HeaderSize(h PixiHeader) int {
	return 2 + len([]byte(d.Name)) + h.OffsetSize + h.OffsetSize
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

// Writes the binary description of the dimenson to the given stream, according to the specification
// in the Pixi header h.
func (d *Dimension) Write(w io.Writer, h PixiHeader) error {
	// write the name, then size and tile size
	err := h.WriteFriendly(w, d.Name)
	if err != nil {
		return err
	}
	err = h.WriteOffset(w, int64(d.Size))
	if err != nil {
		return err
	}
	return h.WriteOffset(w, int64(d.TileSize))
}

// Reads a description of the dimension from the given binary stream, according to the specification
// in the Pixi header h.
func (d *Dimension) Read(r io.Reader, h PixiHeader) error {
	name, err := h.ReadFriendly(r)
	if err != nil {
		return err
	}
	d.Name = name
	size, err := h.ReadOffset(r)
	if err != nil {
		return err
	}
	tileSize, err := h.ReadOffset(r)
	if err != nil {
		return err
	}
	d.Size = int(size)
	d.TileSize = int(tileSize)
	return nil
}

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
// the index increments for the size of the first dimension, then the second (nesting the first), then
// the third (nesting the second (nesting the first)), and so on.
func (set DimensionSet) SampleCoordinates() iter.Seq[SampleCoordinate] {
	return func(yield func(index SampleCoordinate) bool) {
		samples := set.Samples()
		curInd := make(SampleCoordinate, len(set))
		for range samples {
			if !yield(curInd) {
				break
			}
			for dInd := 0; dInd < len(curInd); dInd++ {
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
			for dInd := 0; dInd < len(set); dInd++ {
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
				for dInd := 0; dInd < len(set); dInd++ {
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
