package pixi

import (
	"encoding/binary"
	"math"
)

const (
	PixiFileType string = "pixi"
	PixiVersion  int64  = 1
)

type Summary struct {
	Metadata map[string]string
	Datasets []DataSet
}

type DataSet struct {
	Separated   bool
	Compression Compression
	Dimensions  []Dimension
	Fields      []Field
	TileBytes   []int64
	Offset      int64
}

func (d DataSet) SampleSize() int64 {
	sampleSize := int64(0)
	for _, f := range d.Fields {
		sampleSize += f.Size()
	}
	return sampleSize
}

func (d DataSet) DataSize() int64 {
	size := int64(0)
	for _, b := range d.TileBytes {
		size += b
	}
	return size
}

func (d DataSet) Tiles() int {
	tiles := 0
	for _, t := range d.Dimensions {
		tiles += t.Tiles()
	}
	return tiles
}

func (d DataSet) TileSamples() int64 {
	samples := int64(1)
	for _, d := range d.Dimensions {
		samples *= int64(d.TileSize)
	}
	return samples
}

func (d DataSet) Samples() int64 {
	samples := int64(1)
	for _, dim := range d.Dimensions {
		samples *= dim.Size
	}
	return samples
}

func (d DataSet) TileSize(tileIndex int) int64 {
	if d.Tiles() == 0 {
		return 0
	}
	if d.Separated {
		field := tileIndex / d.Tiles()
		return d.TileSamples() * d.Fields[field].Size()
	} else {
		return d.TileSamples() * d.SampleSize()
	}
}

func (d DataSet) TileOffset(tileIndex int) int64 {
	dataStart := d.Offset
	for i := 0; i < tileIndex; i++ {
		dataStart += d.TileBytes[tileIndex]
	}
	return dataStart
}

type Dimension struct {
	Size     int64
	TileSize int32
}

// Returns the number of tiles in this dimension.
// The number of tiles is calculated by dividing the size of the dimension by the tile size,
// and then rounding up to the nearest whole number if there are any remaining bytes that do not fit into a full tile.
func (d Dimension) Tiles() int {
	if d.Size <= 0 {
		return 0
	}
	if d.TileSize <= 0 {
		panic("pixi: Size of dimension > 0 but TileSize set to 0, invalid")
	}
	tiles := int(d.Size / int64(d.TileSize))
	if d.Size%int64(d.TileSize) != 0 {
		tiles += 1
	}
	return tiles
}

type Field struct {
	Name string
	Type FieldType
}

func (f Field) Size() int64 {
	return f.Type.Size()
}

func (f Field) Read(raw []byte) any {
	return f.Type.Read(raw)
}

func (f Field) Write(raw []byte, val any) {
	f.Type.Write(raw, val)
}

type FieldType uint32

const (
	FieldUnknown FieldType = 0
	FieldInt8    FieldType = 1
	FieldUint8   FieldType = 2
	FieldInt16   FieldType = 3
	FieldUint16  FieldType = 4
	FieldInt32   FieldType = 5
	FieldUint32  FieldType = 6
	FieldInt64   FieldType = 7
	FieldUint64  FieldType = 8
	FieldFloat32 FieldType = 9
	FieldFloat64 FieldType = 10
)

// This function returns the size of a field in bytes.
func (f FieldType) Size() int64 {
	switch f {
	case FieldUnknown:
		return 0
	case FieldInt8:
		return 1
	case FieldInt16:
		return 2
	case FieldInt32:
		return 4
	case FieldInt64:
		return 8
	case FieldUint8:
		return 1
	case FieldUint16:
		return 2
	case FieldUint32:
		return 4
	case FieldUint64:
		return 8
	case FieldFloat32:
		return 4
	case FieldFloat64:
		return 8
	default:
		panic("pixi: unsupported field type")
	}
}

// This function reads the value of a given FieldType from the provided raw byte slice.
// The read operation is type-dependent, with each field type having its own specific method
// for reading values. This ensures that the correct data is read and converted into the
// expected format.
func (f FieldType) Read(raw []byte) any {
	switch f {
	case FieldUnknown:
		panic("pixi: tried to read field with unknown size")
	case FieldInt8:
		return int8(raw[0])
	case FieldUint8:
		return raw[0]
	case FieldInt16:
		return int16(binary.BigEndian.Uint16(raw))
	case FieldUint16:
		return binary.BigEndian.Uint16(raw)
	case FieldInt32:
		return int32(binary.BigEndian.Uint32(raw))
	case FieldUint32:
		return binary.BigEndian.Uint32(raw)
	case FieldInt64:
		return int64(binary.BigEndian.Uint64(raw))
	case FieldUint64:
		return binary.BigEndian.Uint64(raw)
	case FieldFloat32:
		return math.Float32frombits(binary.BigEndian.Uint32(raw))
	case FieldFloat64:
		return math.Float64frombits(binary.BigEndian.Uint64(raw))
	default:
		panic("pixi: tried to read unsupported field type")
	}
}

// This function writes a value of any type into bytes according to the specified FieldType.
// The written bytes are stored in the provided byte array. This function will panic if
// the FieldType is unknown or if an unsupported field type is encountered.
func (f FieldType) Write(raw []byte, val any) {
	switch f {
	case FieldUnknown:
		panic("pixi: tried to write field with unknown size")
	case FieldInt8:
		raw[0] = byte(val.(int8))
	case FieldUint8:
		raw[0] = val.(uint8)
	case FieldInt16:
		binary.BigEndian.PutUint16(raw, uint16(val.(int16)))
	case FieldUint16:
		binary.BigEndian.PutUint16(raw, val.(uint16))
	case FieldInt32:
		binary.BigEndian.PutUint32(raw, uint32(val.(int32)))
	case FieldUint32:
		binary.BigEndian.PutUint32(raw, val.(uint32))
	case FieldInt64:
		binary.BigEndian.PutUint64(raw, uint64(val.(int64)))
	case FieldUint64:
		binary.BigEndian.PutUint64(raw, val.(uint64))
	case FieldFloat32:
		binary.BigEndian.PutUint32(raw, math.Float32bits(val.(float32)))
	case FieldFloat64:
		binary.BigEndian.PutUint64(raw, math.Float64bits(val.(float64)))
	default:
		panic("pixi: tried to write unsupported field type")
	}
}

type Compression uint32

const (
	CompressionNone Compression = 0
	CompressionGzip Compression = 1
)
