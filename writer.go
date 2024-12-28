package pixi

import (
	"encoding/binary"
	"fmt"
	"io"
)

func StartPixi(w io.Writer) (*Pixi, error) {
	// write file type
	_, err := w.Write([]byte(FileType))
	if err != nil {
		return nil, err
	}

	// write file version
	_, err = w.Write([]byte(fmt.Sprintf("%04d", Version)))
	if err != nil {
		return nil, err
	}

	return &Pixi{}, nil
}

func WriteLayerHeader(w io.Writer, d DiskLayer) error {
	tiles := d.DiskTiles()
	if tiles != len(d.TileBytes) {
		return FormatError("invalid TileBytes: must have same number of elements as tiles in data set for valid pixi files")
	}
	if tiles != len(d.TileOffsets) {
		return FormatError("invalid TileOffsets: must have same number of elements as tiles in data set for valid pixi files")
	}

	// first four: dimension count, field count, configuration, compression
	err := binary.Write(w, binary.BigEndian, uint32(len(d.Dimensions)))
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, uint32(len(d.Fields)))
	if err != nil {
		return err
	}

	configuration := uint32(0)
	if d.Separated {
		configuration = 1
	}
	err = binary.Write(w, binary.BigEndian, configuration)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.BigEndian, d.Compression)
	if err != nil {
		return err
	}

	// write layer name
	err = binary.Write(w, binary.BigEndian, uint32(len([]byte(d.Name))))
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, []byte(d.Name))
	if err != nil {
		return err
	}

	// write dimension sizes and tile sizes
	for _, dim := range d.Dimensions {
		err = binary.Write(w, binary.BigEndian, int64(dim.Size))
		if err != nil {
			return err
		}
	}
	for _, dim := range d.Dimensions {
		err = binary.Write(w, binary.BigEndian, int64(dim.TileSize))
		if err != nil {
			return err
		}
	}

	// write field types and names
	for _, f := range d.Fields {
		err = binary.Write(w, binary.BigEndian, f.Type)
		if err != nil {
			return err
		}
	}
	for _, f := range d.Fields {
		err = binary.Write(w, binary.BigEndian, uint16(len([]byte(f.Name))))
		if err != nil {
			return err
		}
		err = binary.Write(w, binary.BigEndian, []byte(f.Name))
		if err != nil {
			return err
		}
	}

	// write tile bytes, offsets, and start of next layer
	err = binary.Write(w, binary.BigEndian, d.TileBytes)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, d.TileOffsets)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, d.NextLayerStart)
	if err != nil {
		return err
	}

	return nil
}
