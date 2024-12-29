package pixi

import (
	"encoding/binary"
	"io"
	"strconv"
)

func ReadPixi(r io.ReadSeeker) (Pixi, error) {
	buf := make([]byte, 4)

	// check file type
	_, err := r.Read(buf)
	if err != nil {
		return Pixi{}, err
	}
	fileType := string(buf)
	if fileType != FileType {
		return Pixi{}, FormatError("pixi file marker not found at start of file")
	}

	// check file version
	_, err = r.Read(buf)
	if err != nil {
		return Pixi{}, err
	}
	version, err := strconv.ParseInt(string(buf), 10, 32)
	if err != nil {
		return Pixi{}, err
	}
	if version > Version {
		return Pixi{}, FormatError("reader does not support this version of pixi file")
	}

	offset := FirstLayerOffset

	layers := []*Layer{}
	for offset != 0 {
		layer, err := ReadLayer(r)
		if err != nil {
			return Pixi{}, err
		}
		layers = append(layers, &layer)
		offset = layer.NextLayerStart
		_, err = r.Seek(offset, io.SeekStart)
		if err != nil {
			return Pixi{}, err
		}
	}

	summary := Pixi{
		Layers: layers,
	}

	return summary, nil
}

func ReadLayer(r io.Reader) (Layer, error) {
	summary := Layer{}
	var dimCount, fieldCount, configuration, nameLen uint32
	err := binary.Read(r, binary.BigEndian, &dimCount)
	if err != nil {
		return summary, err
	}
	err = binary.Read(r, binary.BigEndian, &fieldCount)
	if err != nil {
		return summary, err
	}
	err = binary.Read(r, binary.BigEndian, &configuration)
	if err != nil {
		return summary, err
	}
	summary.Separated = configuration != 0
	err = binary.Read(r, binary.BigEndian, &summary.Compression)
	if err != nil {
		return summary, err
	}

	// read layer name
	err = binary.Read(r, binary.BigEndian, &nameLen)
	if err != nil {
		return summary, err
	}
	name := make([]byte, nameLen)
	_, err = r.Read(name)
	if err != nil {
		return summary, err
	}
	summary.Name = string(name)

	// read dimension sizes
	dimSizes := make([]int64, dimCount)
	err = binary.Read(r, binary.BigEndian, dimSizes)
	if err != nil {
		return summary, err
	}

	// read dimension tile sizes
	tileSizes := make([]int64, dimCount)
	err = binary.Read(r, binary.BigEndian, tileSizes)
	if err != nil {
		return summary, err
	}

	dims := make([]Dimension, dimCount)
	for i := 0; i < int(dimCount); i++ {
		dims[i] = Dimension{Size: int(dimSizes[i]), TileSize: int(tileSizes[i])}
	}
	summary.Dimensions = dims

	// read field types
	fieldTypes := make([]FieldType, fieldCount)
	err = binary.Read(r, binary.BigEndian, fieldTypes)
	if err != nil {
		return summary, err
	}

	// read field names
	fieldNames := make([]string, fieldCount)
	for i := 0; i < int(fieldCount); i++ {
		var nameLen uint16
		err := binary.Read(r, binary.BigEndian, &nameLen)
		if err != nil {
			return summary, err
		}
		buf := make([]byte, nameLen)
		_, err = r.Read(buf)
		if err != nil {
			return summary, err
		}
		fieldNames[i] = string(buf)
	}

	fields := make([]Field, fieldCount)
	for i := 0; i < int(fieldCount); i++ {
		fields[i] = Field{Name: fieldNames[i], Type: fieldTypes[i]}
	}
	summary.Fields = fields

	// read tile bytes, offsets, and next layer start
	tiles := summary.DiskTiles()
	tileBytes := make([]int64, tiles)
	err = binary.Read(r, binary.BigEndian, tileBytes)
	if err != nil {
		return summary, err
	}
	tileOffsets := make([]int64, tiles)
	err = binary.Read(r, binary.BigEndian, tileOffsets)
	if err != nil {
		return summary, err
	}
	var nextLayerStart int64
	err = binary.Read(r, binary.BigEndian, &nextLayerStart)
	if err != nil {
		return summary, err
	}

	summary.TileBytes = tileBytes
	summary.TileOffsets = tileOffsets
	summary.NextLayerStart = nextLayerStart

	return summary, nil
}
