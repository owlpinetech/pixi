package pixi

import (
	"bytes"
	"encoding/binary"
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

func (d Dimension) Tiles() int {
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

func (f Field) Read(raw []byte) (any, error) {
	return f.Type.Read(raw)
}

func (f Field) Write(raw []byte, val any) error {
	return f.Type.Write(raw, val)
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

func (f FieldType) Read(raw []byte) (any, error) {
	switch f {
	case FieldUnknown:
		panic("pixi: tried to read field with unknown size")
	case FieldInt8:
		return int8(raw[0]), nil
	case FieldUint8:
		return raw[0], nil
	case FieldInt16:
		var res int16
		err := binary.Read(bytes.NewReader(raw), binary.BigEndian, &res)
		return res, err
	case FieldUint16:
		return binary.BigEndian.Uint16(raw), nil
	case FieldInt32:
		var res int32
		err := binary.Read(bytes.NewReader(raw), binary.BigEndian, &res)
		return res, err
	case FieldUint32:
		return binary.BigEndian.Uint32(raw), nil
	case FieldInt64:
		var res int64
		err := binary.Read(bytes.NewReader(raw), binary.BigEndian, &res)
		return res, err
	case FieldUint64:
		return binary.BigEndian.Uint64(raw), nil
	case FieldFloat32:
		var res float32
		err := binary.Read(bytes.NewReader(raw), binary.BigEndian, &res)
		return res, err
	case FieldFloat64:
		var res float64
		err := binary.Read(bytes.NewReader(raw), binary.BigEndian, &res)
		return res, err
	default:
		panic("pixi: tried to read unsupported field type")
	}
}

func (f FieldType) Write(raw []byte, val any) error {
	switch f {
	case FieldUnknown:
		panic("pixi: tried to write field with unknown size")
	case FieldInt8:
		raw[0] = byte(val.(int8))
		return nil
	case FieldUint8:
		raw[0] = val.(uint8)
		return nil
	case FieldInt16:
		return binary.Write(bytes.NewBuffer(raw), binary.BigEndian, val.(int16))
	case FieldUint16:
		binary.BigEndian.PutUint16(raw, val.(uint16))
		return nil
	case FieldInt32:
		return binary.Write(bytes.NewBuffer(raw), binary.BigEndian, val.(int32))
	case FieldUint32:
		binary.BigEndian.PutUint32(raw, val.(uint32))
		return nil
	case FieldInt64:
		return binary.Write(bytes.NewBuffer(raw), binary.BigEndian, val.(int64))
	case FieldUint64:
		binary.BigEndian.PutUint64(raw, val.(uint64))
		return nil
	case FieldFloat32:
		return binary.Write(bytes.NewBuffer(raw), binary.BigEndian, val.(float32))
	case FieldFloat64:
		return binary.Write(bytes.NewBuffer(raw), binary.BigEndian, val.(float64))
	default:
		panic("pixi: tried to write unsupported field type")
	}
}

type Compression uint32

const (
	CompressionNone Compression = 0
	CompressionGzip Compression = 1
)
