package pixi

import (
	"io"
	"strconv"
)

// Represents an axis along which tiled, gridded data is stored in a Pixi file. Data sets can have
// one or more dimensions, but never zero. If a dimension is not tiled, then the TileSize should be
// the same as a the total Size.
type Dimension struct {
	Name     string // Friendly name to refer to the dimension in the layer.
	Size     int    // The total number of elements in the dimension.
	TileSize int    // The size of the tiles in the dimension. Does not need to be a factor of Size.
}

// Get the size in bytes of this dimension description as it is laid out and written to disk.
func (d Dimension) HeaderSize(h Header) int {
	return 2 + len([]byte(d.Name)) + 2*int(h.OffsetSize)
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
func (d Dimension) Write(w io.Writer, h Header) error {
	if d.Size <= 0 || d.TileSize <= 0 {
		return ErrFormat("dimension size and tile size must be greater than 0")
	}
	if d.Size < d.TileSize {
		return ErrFormat("dimension tile size cannot be larger than dimension total size")
	}
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
func (d *Dimension) Read(r io.Reader, h Header) error {
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

func (d Dimension) String() string {
	return d.Name + "(" + strconv.Itoa(d.Size) + " / " + strconv.Itoa(d.TileSize) + ")"
}
