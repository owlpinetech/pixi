package pixi

import (
	"fmt"
	"io"
)

const (
	FileType         string = "pixi" // Every file starts with these four bytes.
	Version          int64  = 1      // Every file has a version number as the second set of four bytes.
	FirstLayerOffset int64  = 8      // The byte offset in every Pixi file at which starts the description of the first accessible layer
	OffsetUnset      int64  = -1
)

// Represents a single pixi file composed of one or more layers. Functions as a handle
// to access the description of the each layer as well as the data stored in each layer.
type Pixi struct {
	Layers []*DiskLayer // The metadata information about each layer in the file.
}

func (d *Pixi) LayerOffset(l *DiskLayer) int64 {
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

// Squishes layers together so there are no un-used bytes between them in the backing file.
func (d *Pixi) Compact(backing io.ReadWriteSeeker) error {
	return fmt.Errorf("unimplemented")
}

func (d *Pixi) AddLayer(backing io.ReadWriteSeeker, offset int64, layer Layer) (*DiskLayer, error) {
	// seek to layer start
	_, err := backing.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	// set tile sizes and offsets to special 'unset' value
	diskLayer := &DiskLayer{Layer: layer}
	diskLayer.TileBytes = make([]int64, layer.DiskTiles())
	diskLayer.TileOffsets = make([]int64, layer.DiskTiles())
	for i := range diskLayer.TileBytes {
		diskLayer.TileBytes[i] = 0
		diskLayer.TileOffsets[i] = OffsetUnset
	}

	// write the layer header
	if err := WriteLayerHeader(backing, *diskLayer); err != nil {
		return nil, err
	}

	// write offset of previous layer last in case writes to current layer failed
	if err := d.overwriteLastOffset(backing, offset); err != nil {
		return nil, err
	}

	d.Layers = append(d.Layers, diskLayer)

	return d.Layers[len(d.Layers)-1], nil
}

func (d *Pixi) AddBlankUncompressedLayer(backing io.ReadWriteSeeker, offset int64, layer Layer) (*DiskLayer, error) {
	// seek to layer start
	_, err := backing.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	// compute tile sizes and offsets (easy since uncompressed)
	diskLayer := &DiskLayer{Layer: layer}
	diskLayer.Compression = CompressionNone
	diskLayer.TileBytes = make([]int64, layer.DiskTiles())
	diskLayer.TileOffsets = make([]int64, layer.DiskTiles())

	tileOffset := offset + diskLayer.DiskHeaderSize()
	for i := range diskLayer.TileBytes {
		tileSize := int64(diskLayer.DiskTileSize(i))
		diskLayer.TileBytes[i] = tileSize
		diskLayer.TileOffsets[i] = tileOffset
		tileOffset += tileSize
	}

	// write the layer header
	if err := WriteLayerHeader(backing, *diskLayer); err != nil {
		return nil, err
	}

	// write each tile in the layer, accounting for separated vs contiguous
	buf := make([]byte, 0)
	for i := 0; i < layer.DiskTiles(); i++ {
		tileSize := layer.DiskTileSize(i)
		if tileSize != len(buf) {
			buf = make([]byte, tileSize)
		}
		if _, err := backing.Write(buf); err != nil {
			return nil, err
		}
	}

	// write offset of previous layer last in case writes to current layer failed
	if err := d.overwriteLastOffset(backing, offset); err != nil {
		return nil, err
	}

	d.Layers = append(d.Layers, diskLayer)

	return d.Layers[len(d.Layers)-1], nil
}

func (d *Pixi) overwriteLastOffset(backing io.ReadWriteSeeker, offset int64) error {
	if len(d.Layers) > 0 {
		d.Layers[len(d.Layers)-1].NextLayerStart = offset
		if err := WriteLayerHeader(backing, *d.Layers[len(d.Layers)-1]); err != nil {
			d.Layers[len(d.Layers)-1].NextLayerStart = 0
			return err
		}
	}
	return nil
}
