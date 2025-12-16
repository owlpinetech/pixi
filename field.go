package pixi

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/kshard/float8"
	"github.com/x448/float16"
)

// Describes a set of values in a data set with a common shape. Similar to a field of a record
// in a database, but with a more restricted set of available types per field.
type Field struct {
	Name string    // A friendly name for this field, to help guide interpretation of the data.
	Type FieldType // The type of data stored in each element of this field.
	Min  any       // Optional minimum value for the range of data in this field. Must match Type if present.
	Max  any       // Optional maximum value for the range of data in this field. Must match Type if present.
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
func (f Field) ValueToBytes(val any, order binary.ByteOrder, raw []byte) {
	f.Type.ValueToBytes(val, order, raw)
}

// Get the size in bytes of this dimension description as it is laid out and written to disk.
func (d Field) HeaderSize(h *PixiHeader) int {
	size := 2 + len([]byte(d.Name)) + 4 // base size: name + field type

	// Add size for optional Min value
	if d.Min != nil {
		size += d.Type.Base().Size()
	}

	// Add size for optional Max value
	if d.Max != nil {
		size += d.Type.Base().Size()
	}

	return size
}

// Writes the binary description of the field to the given stream, according to the specification
// in the Pixi header h.
func (d Field) Write(w io.Writer, h *PixiHeader) error {
	// Set flags based on presence of Min/Max values
	encodedType := d.Type.WithMin(d.Min != nil).WithMax(d.Max != nil)

	// write the name, then the field type with flags
	err := h.WriteFriendly(w, d.Name)
	if err != nil {
		return err
	}

	err = h.Write(w, encodedType)
	if err != nil {
		return err
	}

	// Write optional Min value
	if d.Min != nil {
		minBytes := make([]byte, d.Type.Base().Size())
		d.Type.Base().ValueToBytes(d.Min, h.ByteOrder, minBytes)
		_, err = w.Write(minBytes)
		if err != nil {
			return err
		}
	}

	// Write optional Max value
	if d.Max != nil {
		maxBytes := make([]byte, d.Type.Base().Size())
		d.Type.Base().ValueToBytes(d.Max, h.ByteOrder, maxBytes)
		_, err = w.Write(maxBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

// Reads a description of the field from the given binary stream, according to the specification
// in the Pixi header h.
func (d *Field) Read(r io.Reader, h *PixiHeader) error {
	name, err := h.ReadFriendly(r)
	if err != nil {
		return err
	}
	d.Name = name

	var encodedType FieldType
	err = h.Read(r, &encodedType)
	if err != nil {
		return err
	}

	// Extract base type and flags
	d.Type = encodedType.Base()

	// Read optional Min value
	if encodedType.HasMin() {
		minBytes := make([]byte, d.Type.Size())
		_, err = r.Read(minBytes)
		if err != nil {
			return err
		}
		d.Min = d.Type.BytesToValue(minBytes, h.ByteOrder)
	} else {
		d.Min = nil
	}

	// Read optional Max value
	if encodedType.HasMax() {
		maxBytes := make([]byte, d.Type.Size())
		_, err = r.Read(maxBytes)
		if err != nil {
			return err
		}
		d.Max = d.Type.BytesToValue(maxBytes, h.ByteOrder)
	} else {
		d.Max = nil
	}

	return nil
}

// Updates the field's Min and Max values based on a new value. Returns true if the field was modified.
func (field *Field) UpdateMinMax(value any) bool {
	changed := false

	// Update Min if needed
	if field.Min == nil || field.CompareValues(value, field.Min) < 0 {
		field.Min = value
		changed = true
	}

	// Update Max if needed
	if field.Max == nil || field.CompareValues(value, field.Max) > 0 {
		field.Max = value
		changed = true
	}

	return changed
}

// Compares two values based on the field type. Returns -1 if a < b, 0 if a == b, 1 if a > b.
func (field *Field) CompareValues(a, b any) int {
	switch field.Type.Base() {
	case FieldInt8:
		va, vb := a.(int8), b.(int8)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case FieldUint8:
		va, vb := a.(uint8), b.(uint8)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case FieldInt16:
		va, vb := a.(int16), b.(int16)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case FieldUint16:
		va, vb := a.(uint16), b.(uint16)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case FieldInt32:
		va, vb := a.(int32), b.(int32)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case FieldUint32:
		va, vb := a.(uint32), b.(uint32)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case FieldInt64:
		va, vb := a.(int64), b.(int64)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case FieldUint64:
		va, vb := a.(uint64), b.(uint64)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case FieldFloat8:
		va, vb := float64(a.(float8.Float8)), float64(b.(float8.Float8))
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case FieldFloat16:
		va, vb := a.(float16.Float16).Float32(), b.(float16.Float16).Float32()
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case FieldFloat32:
		va, vb := a.(float32), b.(float32)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case FieldFloat64:
		va, vb := a.(float64), b.(float64)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case FieldBool:
		va, vb := a.(bool), b.(bool)
		if !va && vb {
			return -1
		}
		if va && !vb {
			return 1
		}
		return 0
	default:
		return 0
	}
}

// Describes the size and interpretation of a field.
type FieldType uint32

const (
	fieldTypeBaseMask FieldType = 0x3FFFFFFF // Mask for the base field type (lower 30 bits)
	fieldTypeMinFlag  FieldType = 0x40000000 // Flag for Min value presence (bit 30)
	fieldTypeMaxFlag  FieldType = 0x80000000 // Flag for Max value presence (bit 31)
)

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
	FieldFloat8  FieldType = 9  // An 8-bit floating point number.
	FieldFloat16 FieldType = 10 // A 16-bit floating point number.
	FieldFloat32 FieldType = 11 // A 32-bit floating point number.
	FieldFloat64 FieldType = 12 // A 64-bit floating point number.
	FieldBool    FieldType = 13 // A boolean value.
)

// Returns the base field type without the optional flags.
func (f FieldType) Base() FieldType {
	return f & fieldTypeBaseMask
}

// Returns whether the Min value flag is set.
func (f FieldType) HasMin() bool {
	return f&fieldTypeMinFlag != 0
}

// Returns whether the Max value flag is set.
func (f FieldType) HasMax() bool {
	return f&fieldTypeMaxFlag != 0
}

// Returns a new FieldType with the Min flag set or cleared.
func (f FieldType) WithMin(hasMin bool) FieldType {
	if hasMin {
		return f | fieldTypeMinFlag
	}
	return f & ^fieldTypeMinFlag
}

// Returns a new FieldType with the Max flag set or cleared.
func (f FieldType) WithMax(hasMax bool) FieldType {
	if hasMax {
		return f | fieldTypeMaxFlag
	}
	return f & ^fieldTypeMaxFlag
}

// This function returns the size of each element in a field in bytes.
func (f FieldType) Size() int {
	switch f.Base() {
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
	case FieldFloat8:
		return 1
	case FieldFloat16:
		return 2
	case FieldFloat32:
		return 4
	case FieldFloat64:
		return 8
	case FieldBool:
		return 1
	default:
		panic("pixi: unsupported field type")
	}
}

func (f FieldType) String() string {
	switch f.Base() {
	case FieldUnknown:
		return "unknown"
	case FieldInt8:
		return "int8"
	case FieldInt16:
		return "int16"
	case FieldInt32:
		return "int32"
	case FieldInt64:
		return "int64"
	case FieldUint8:
		return "uint8"
	case FieldUint16:
		return "uint16"
	case FieldUint32:
		return "uint32"
	case FieldUint64:
		return "uint64"
	case FieldFloat8:
		return "float8"
	case FieldFloat16:
		return "float16"
	case FieldFloat32:
		return "float32"
	case FieldFloat64:
		return "float64"
	case FieldBool:
		return "bool"
	default:
		panic("pixi: unsupported field type")
	}
}

// This function reads the value of a given FieldType from the provided raw byte slice.
// The read operation is type-dependent, with each field type having its own specific method
// for reading values. This ensures that the correct data is read and converted into the
// expected format.
func (f FieldType) BytesToValue(raw []byte, o binary.ByteOrder) any {
	switch f.Base() {
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
	case FieldFloat8:
		return float8.Float8(uint8(raw[0]))
	case FieldFloat16:
		return float16.Frombits(o.Uint16(raw))
	case FieldFloat32:
		return math.Float32frombits(o.Uint32(raw))
	case FieldFloat64:
		return math.Float64frombits(o.Uint64(raw))
	case FieldBool:
		return raw[0] != 0
	default:
		panic("pixi: tried to read unsupported field type")
	}
}

// Writes the given value, assumed to correspond to the FieldType, into it's raw representation
// in bytes according to the byte order specified.
func (f FieldType) ValueToBytes(val any, o binary.ByteOrder, bytes []byte) {
	switch f.Base() {
	case FieldUnknown:
		panic("pixi: tried to write field with unknown size")
	case FieldInt8:
		bytes[0] = byte(val.(int8))
	case FieldUint8:
		bytes[0] = val.(uint8)
	case FieldInt16:
		o.PutUint16(bytes, uint16(val.(int16)))
	case FieldUint16:
		o.PutUint16(bytes, val.(uint16))
	case FieldInt32:
		o.PutUint32(bytes, uint32(val.(int32)))
	case FieldUint32:
		o.PutUint32(bytes, val.(uint32))
	case FieldInt64:
		o.PutUint64(bytes, uint64(val.(int64)))
	case FieldUint64:
		o.PutUint64(bytes, val.(uint64))
	case FieldFloat8:
		bytes[0] = byte(val.(float8.Float8))
	case FieldFloat16:
		o.PutUint16(bytes, val.(float16.Float16).Bits())
	case FieldFloat32:
		o.PutUint32(bytes, math.Float32bits(val.(float32)))
	case FieldFloat64:
		o.PutUint64(bytes, math.Float64bits(val.(float64)))
	case FieldBool:
		if val.(bool) {
			bytes[0] = 1
		} else {
			bytes[0] = 0
		}
	default:
		panic("pixi: tried to write unsupported field type")
	}
}
