package pixi

import (
	"encoding/binary"
	"fmt"
	"io"
)

func WriteSummary(w io.Writer, s Summary) error {
	// write file type
	_, err := w.Write([]byte(PixiFileType))
	if err != nil {
		return err
	}

	// write file version
	_, err = w.Write([]byte(fmt.Sprintf("%04d", PixiVersion)))
	if err != nil {
		return err
	}

	// write all metadata strings
	err = binary.Write(w, binary.BigEndian, uint32(len(s.Metadata)))
	if err != nil {
		return err
	}
	for k, v := range s.Metadata {
		err = WriteMetadata(w, k, v)
		if err != nil {
			return err
		}
	}

	// write all dataset headers
	err = WriteFixedSummary(w, s)
	if err != nil {
		return err
	}

	return nil
}

func WriteMetadata(w io.Writer, key string, val string) error {
	// write key string
	err := binary.Write(w, binary.BigEndian, uint32(len(key)))
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.BigEndian, []byte(key))
	if err != nil {
		return err
	}

	// write value string
	err = binary.Write(w, binary.BigEndian, uint32(len(val)))
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.BigEndian, []byte(val))
	if err != nil {
		return err
	}

	return nil
}

func WriteFixedSummary(w io.Writer, d Summary) error {
	tiles := d.Tiles()
	if d.Separated {
		tiles *= len(d.Fields)
	}
	if tiles != len(d.TileBytes) {
		return FormatError("invalid TileBytes: must have same number of elements as tiles in data set for valid pixi files")
	}

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

	// write dimension sizes and tile sizes
	for _, dim := range d.Dimensions {
		err = binary.Write(w, binary.BigEndian, dim.Size)
		if err != nil {
			return err
		}
	}
	for _, dim := range d.Dimensions {
		err = binary.Write(w, binary.BigEndian, dim.TileSize)
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
		err = binary.Write(w, binary.BigEndian, uint16(len(f.Name)))
		if err != nil {
			return err
		}
		err = binary.Write(w, binary.BigEndian, []byte(f.Name))
		if err != nil {
			return err
		}
	}

	// write tile bytes
	err = binary.Write(w, binary.BigEndian, d.TileBytes)
	if err != nil {
		return err
	}

	return nil
}
