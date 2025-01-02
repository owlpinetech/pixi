package pixi

import (
	"encoding/binary"
	"io"
	"math"
)

// Describes a set of values in a data set with a common shape. Similar to a field of a record
// in a database, but with a more restricted set of available types per field.
type Field struct {
	Name string    // A friendly name for this field, to help guide interpretation of the data.
	Type FieldType // The type of data stored in each element of this field.
}

// Returns the size of a field in bytes.
func (f Field) Size() int {
	return f.Type.Size()
}

// Reads the value of a given FieldType from the provided raw byte slice.
// The read operation is type-dependent, with each field type having its own specific method
// for reading values. This ensures that the correct data is read and converted into the
// expected format.
func (f Field) BytesToValue(raw []byte, order binary.ByteOrder) any {
	return f.Type.BytesToValue(raw, order)
}

// This function writes a value of any type into bytes according to the specified FieldType.
// The written bytes are stored in the provided byte array. This function will panic if
// the FieldType is unknown or if an unsupported field type is encountered.
func (f Field) WriteValue(raw []byte, val any) {
	f.Type.WriteValue(raw, val)
}

// Get the size in bytes of this dimension description as it is laid out and written to disk.
func (d Field) HeaderSize(h PixiHeader) int {
	return 2 + len([]byte(d.Name)) + 4
}

// Writes the binary description of the field to the given stream, according to the specification
// in the Pixi header h.
func (d *Field) Write(w io.Writer, h PixiHeader) error {
	// write the name, then size and tile size
	err := h.WriteFriendly(w, d.Name)
	if err != nil {
		return err
	}
	return h.Write(w, d.Type)
}

// Reads a description of the field from the given binary stream, according to the specification
// in the Pixi header h.
func (d *Field) Read(r io.Reader, h PixiHeader) error {
	name, err := h.ReadFriendly(r)
	if err != nil {
		return err
	}
	d.Name = name
	return h.Read(r, &d.Type)
}

// Describes the size and interpretation of a field.
type FieldType uint32

const (
	FieldUnknown FieldType = 0  // Generally indicates an error.
	FieldInt8    FieldType = 1  // An 8-bit signed integer.
	FieldUint8   FieldType = 2  // An 8-bit unsigned integer.
	FieldInt16   FieldType = 3  // A 16-bit signed integer.
	FieldUint16  FieldType = 4  // A 16-bit unsigned integer.
	FieldInt32   FieldType = 5  // A 32-bit signed integer.
	FieldUint32  FieldType = 6  // A 32-bit unsigned integer.
	FieldInt64   FieldType = 7  // A 64-bit signed integer.
	FieldUint64  FieldType = 8  // A 64-bit unsigned integer.
	FieldFloat32 FieldType = 9  // A 32-bit floating point number.
	FieldFloat64 FieldType = 10 // A 64-bit floating point number.
)

// This function returns the size of each element in a field in bytes.
func (f FieldType) Size() int {
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

func (f FieldType) ReadValue(r io.Reader, o binary.ByteOrder) (any, error) {
	switch f {
	case FieldUnknown:
		panic("pixi: tried to read field with unknown size")
	case FieldInt8:
		var val int8
		err := binary.Read(r, o, &val)
		return val, err
	case FieldUint8:
		var val uint8
		err := binary.Read(r, o, &val)
		return val, err
	case FieldInt16:
		var val int16
		err := binary.Read(r, o, &val)
		return val, err
	case FieldUint16:
		var val uint16
		err := binary.Read(r, o, &val)
		return val, err
	case FieldInt32:
		var val int32
		err := binary.Read(r, o, &val)
		return val, err
	case FieldUint32:
		var val uint32
		err := binary.Read(r, o, &val)
		return val, err
	case FieldInt64:
		var val int64
		err := binary.Read(r, o, &val)
		return val, err
	case FieldUint64:
		var val uint64
		err := binary.Read(r, o, &val)
		return val, err
	case FieldFloat32:
		var val float32
		err := binary.Read(r, o, &val)
		return val, err
	case FieldFloat64:
		var val float64
		err := binary.Read(r, o, &val)
		return val, err
	default:
		panic("pixi: tried to read unsupported field type")
	}
}

// This function reads the value of a given FieldType from the provided raw byte slice.
// The read operation is type-dependent, with each field type having its own specific method
// for reading values. This ensures that the correct data is read and converted into the
// expected format.
func (f FieldType) BytesToValue(raw []byte, o binary.ByteOrder) any {
	switch f {
	case FieldUnknown:
		panic("pixi: tried to read field with unknown size")
	case FieldInt8:
		return int8(raw[0])
	case FieldUint8:
		return raw[0]
	case FieldInt16:
		return int16(o.Uint16(raw))
	case FieldUint16:
		return o.Uint16(raw)
	case FieldInt32:
		return int32(o.Uint32(raw))
	case FieldUint32:
		return o.Uint32(raw)
	case FieldInt64:
		return int64(o.Uint64(raw))
	case FieldUint64:
		return o.Uint64(raw)
	case FieldFloat32:
		return math.Float32frombits(o.Uint32(raw))
	case FieldFloat64:
		return math.Float64frombits(o.Uint64(raw))
	default:
		panic("pixi: tried to read unsupported field type")
	}
}

// This function writes a value of any type into bytes according to the specified FieldType.
// The written bytes are stored in the provided byte array. This function will panic if
// the FieldType is unknown or if an unsupported field type is encountered.
func (f FieldType) WriteValue(raw []byte, val any) {
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
