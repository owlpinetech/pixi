package pixi

import (
	"encoding/binary"
	"io"
	"strconv"
)

func ReadSummary(r io.ReadSeeker) (Summary, error) {
	buf := make([]byte, 4)

	// check file type
	_, err := r.Read(buf)
	if err != nil {
		return Summary{}, err
	}
	fileType := string(buf)
	if fileType != PixiFileType {
		return Summary{}, FormatError("pixi file marker not found at start of file")
	}

	// check file version
	_, err = r.Read(buf)
	if err != nil {
		return Summary{}, err
	}
	version, err := strconv.ParseInt(string(buf), 10, 32)
	if err != nil {
		return Summary{}, err
	}
	if version > PixiVersion {
		return Summary{}, FormatError("reader does not support this version of pixi file")
	}

	// read all metadata strings
	var metadataCount uint32
	err = binary.Read(r, binary.BigEndian, &metadataCount)
	if err != nil {
		return Summary{}, err
	}
	metadata := make(map[string]string, metadataCount)
	for i := 0; i < int(metadataCount); i++ {
		key, val, err := ReadMetadata(r)
		if err != nil {
			return Summary{}, err
		}
		metadata[key] = val
	}

	// read the fixed portion of the dataset summary
	summary, err := ReadFixedSummary(r)
	if err != nil {
		return Summary{}, err
	}
	summary.Metadata = metadata

	return summary, nil
}

func ReadMetadata(r io.Reader) (string, string, error) {
	// read string key
	var keyCount uint32
	err := binary.Read(r, binary.BigEndian, &keyCount)
	if err != nil {
		return "", "", err
	}
	key := make([]byte, keyCount)
	err = binary.Read(r, binary.BigEndian, key)
	if err != nil {
		return "", "", err
	}

	// read string value
	var valCount uint32
	err = binary.Read(r, binary.BigEndian, &valCount)
	if err != nil {
		return string(key), "", err
	}
	val := make([]byte, valCount)
	err = binary.Read(r, binary.BigEndian, val)
	if err != nil {
		return string(key), "", err
	}

	return string(key), string(val), nil
}

func ReadFixedSummary(r io.Reader) (Summary, error) {
	summary := Summary{}
	var dimCount, fieldCount, configuration uint32
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
	err = binary.Read(r, binary.BigEndian, &summary.Offset)
	if err != nil {
		return summary, err
	}

	// read dimension sizes
	dimSizes := make([]int64, dimCount)
	err = binary.Read(r, binary.BigEndian, dimSizes)
	if err != nil {
		return summary, err
	}

	// read dimension tile sizes
	tileSizes := make([]int32, dimCount)
	err = binary.Read(r, binary.BigEndian, tileSizes)
	if err != nil {
		return summary, err
	}

	dims := make([]Dimension, dimCount)
	for i := 0; i < int(dimCount); i++ {
		dims[i] = Dimension{Size: dimSizes[i], TileSize: tileSizes[i]}
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

	// read tile bytes
	tiles := summary.Tiles()
	if summary.Separated {
		tiles *= int(fieldCount)
	}
	tileBytes := make([]int64, tiles)
	err = binary.Read(r, binary.BigEndian, tileBytes)
	if err != nil {
		return summary, err
	}
	summary.TileBytes = tileBytes

	return summary, nil
}
