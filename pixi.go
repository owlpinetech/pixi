package pixi

import "io"

const (
	PixiFileType string = "pixi" // Every file starts with these four bytes.
	PixiVersion  int64  = 1      // Every file has a version number as the second set of four bytes.
)

// Information about how data is stored and organized for a particular data set
// inside a pixi file.
type Pixi struct {
	Layers []*DiskLayer // The metadata information about each layer in the file.
}

func (d *Pixi) LayerOffset(l *DiskLayer) int64 {
	offset := d.FirstLayerOffset()
	for _, item := range d.Layers {
		if item == l {
			break
		}
		offset = item.NextLayerStart
	}
	return offset
}

func (d *Pixi) FirstLayerOffset() int64 {
	// four for version, four for file type sequences
	return 8
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
		tileSize := int64(diskLayer.TileSize(i))
		diskLayer.TileBytes[i] = tileSize
		diskLayer.TileOffsets[i] = tileOffset
		tileOffset += tileSize
	}

	// write the layer header
	if err := WriteLayer(backing, *diskLayer); err != nil {
		return nil, err
	}

	// write each tile in the layer, accounting for separated vs contiguous
	buf := make([]byte, 0)
	for i := 0; i < layer.DiskTiles(); i++ {
		tileSize := layer.TileSize(i)
		if tileSize != len(buf) {
			buf = make([]byte, tileSize)
		}
		if _, err := backing.Write(buf); err != nil {
			return nil, err
		}
	}

	// write offset of previous layer last in case writes to current layer failed
	if len(d.Layers) > 0 {
		d.Layers[len(d.Layers)-1].NextLayerStart = offset
		if err := WriteLayer(backing, *d.Layers[len(d.Layers)-1]); err != nil {
			d.Layers[len(d.Layers)-1].NextLayerStart = 0
			return nil, err
		}
	}

	d.Layers = append(d.Layers, diskLayer)

	return d.Layers[len(d.Layers)-1], nil
}

type Compression uint32

const (
	CompressionNone  Compression = 0
	CompressionFlate Compression = 1
)
