package pixi

const (
	FileType         string = "pixi" // Every file starts with these four bytes.
	Version          int64  = 1      // Every file has a version number as the second set of four bytes.
	FirstLayerOffset int64  = 8      // The byte offset in every Pixi file at which starts the description of the first accessible layer
	OffsetUnset      int64  = -1
)

// Represents a single pixi file composed of one or more layers. Functions as a handle
// to access the description of the each layer as well as the data stored in each layer.
type Pixi struct {
	Layers []*Layer // The metadata information about each layer in the file.
}

func (d *Pixi) LayerOffset(l *Layer) int64 {
	offset := FirstLayerOffset
	for _, item := range d.Layers {
		if item == l {
			break
		}
		offset = item.NextLayerStart
	}
	return offset
}

// The total size of the data portions of the file in bytes. Does not count header information
// as part of the size.
func (d *Pixi) DiskDataBytes() int64 {
	size := int64(0)
	for _, l := range d.Layers {
		for _, t := range l.TileBytes {
			size += t
		}
	}
	return size
}
