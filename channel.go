package gopixi

import (
	"cmp"
	"encoding/binary"
	"io"
	"math"

	"github.com/chenxingqiang/go-floatx"
	"github.com/kshard/float8"
	"github.com/shogo82148/float128"
	"github.com/shogo82148/int128"
	"github.com/x448/float16"
)

// Describes a set of values in a data set with a common shape. Similar to a column of a table
// in a database, but with a more restricted set of available types per channel.
type Channel struct {
	Name string      // A friendly name for this channel, to help guide interpretation of the data.
	Type ChannelType // The type of data stored in each element of this channel.
	Min  any         // Optional minimum value for the range of data in this channel. Must match Type if present.
	Max  any         // Optional maximum value for the range of data in this channel. Must match Type if present.
}

// Returns the size of a channel in bytes.
func (c Channel) Size() int {
	return c.Type.Size()
}

// Reads the value of a given ChannelType from the provided raw byte slice.
// The read operation is type-dependent, with each channel type having its own specific method
// for reading values. This ensures that the correct data is read and converted into the
// expected format.
func (c Channel) Value(raw []byte, order binary.ByteOrder) any {
	return c.Type.Value(raw, order)
}

// This function writes a value of any type into bytes according to the specified ChannelType.
// The written bytes are stored in the provided byte array. This function will panic if
// the ChannelType is unknown or if an unsupported channel type is encountered.
func (c Channel) PutValue(val any, order binary.ByteOrder, raw []byte) {
	c.Type.PutValue(val, order, raw)
}

// Get the size in bytes of this dimension description as it is laid out and written to disk.
func (c Channel) HeaderSize(h Header) int {
	size := 2 + len([]byte(c.Name)) + 4 // base size: name + channel type

	// Add size for optional Min value
	if c.Min != nil {
		size += c.Type.Base().Size()
	}

	// Add size for optional Max value
	if c.Max != nil {
		size += c.Type.Base().Size()
	}

	return size
}

// Writes the binary description of the channel to the given stream, according to the specification
// in the Pixi header h.
func (c Channel) Write(w io.Writer, h Header) error {
	// Set flags based on presence of Min/Max values
	encodedType := c.Type.WithMin(c.Min != nil).WithMax(c.Max != nil)

	// write the name, then the channel type with flags
	err := h.WriteFriendly(w, c.Name)
	if err != nil {
		return err
	}

	err = h.Write(w, encodedType)
	if err != nil {
		return err
	}

	// Write optional Min value
	if c.Min != nil {
		minBytes := make([]byte, c.Type.Base().Size())
		c.Type.Base().PutValue(c.Min, h.ByteOrder, minBytes)
		_, err = w.Write(minBytes)
		if err != nil {
			return err
		}
	}

	// Write optional Max value
	if c.Max != nil {
		maxBytes := make([]byte, c.Type.Base().Size())
		c.Type.Base().PutValue(c.Max, h.ByteOrder, maxBytes)
		_, err = w.Write(maxBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

// Reads a description of the channel from the given binary stream, according to the specification
// in the Pixi header h.
func (c *Channel) Read(r io.Reader, h Header) error {
	name, err := h.ReadFriendly(r)
	if err != nil {
		return err
	}
	c.Name = name

	var encodedType ChannelType
	err = h.Read(r, &encodedType)
	if err != nil {
		return err
	}

	// Extract base type and flags
	c.Type = encodedType.Base()

	// Read optional Min value
	if encodedType.HasMin() {
		minBytes := make([]byte, c.Type.Size())
		_, err = r.Read(minBytes)
		if err != nil {
			return err
		}
		c.Min = c.Type.Value(minBytes, h.ByteOrder)
	} else {
		c.Min = nil
	}

	// Read optional Max value
	if encodedType.HasMax() {
		maxBytes := make([]byte, c.Type.Size())
		_, err = r.Read(maxBytes)
		if err != nil {
			return err
		}
		c.Max = c.Type.Value(maxBytes, h.ByteOrder)
	} else {
		c.Max = nil
	}

	return nil
}

// Updates the channel's Min and Max values based on a new value. Returns true if the channel was modified.
func (channel Channel) WithMinMax(value any) Channel {
	// Update Min if needed
	if channel.Min == nil || channel.Type.CompareValues(value, channel.Min) < 0 {
		channel.Min = value
	}

	// Update Max if needed
	if channel.Max == nil || channel.Type.CompareValues(value, channel.Max) > 0 {
		channel.Max = value
	}

	return channel
}

// Describes the size and interpretation of a channel.
type ChannelType uint32

const (
	channelTypeBaseMask ChannelType = 0x3FFFFFFF // Mask for the base channel type (lower 30 bits)
	channelTypeMinFlag  ChannelType = 0x40000000 // Flag for Min value presence (bit 30)
	channelTypeMaxFlag  ChannelType = 0x80000000 // Flag for Max value presence (bit 31)
)

const (
	ChannelUnknown  ChannelType = 0  // Generally indicates an error.
	ChannelInt8     ChannelType = 1  // An 8-bit signed integer.
	ChannelUint8    ChannelType = 2  // An 8-bit unsigned integer.
	ChannelInt16    ChannelType = 3  // A 16-bit signed integer.
	ChannelUint16   ChannelType = 4  // A 16-bit unsigned integer.
	ChannelInt32    ChannelType = 5  // A 32-bit signed integer.
	ChannelUint32   ChannelType = 6  // A 32-bit unsigned integer.
	ChannelInt64    ChannelType = 7  // A 64-bit signed integer.
	ChannelUint64   ChannelType = 8  // A 64-bit unsigned integer.
	ChannelFloat8   ChannelType = 9  // An 8-bit floating point number.
	ChannelFloat16  ChannelType = 10 // A 16-bit floating point number.
	ChannelFloat32  ChannelType = 11 // A 32-bit floating point number.
	ChannelFloat64  ChannelType = 12 // A 64-bit floating point number.
	ChannelBool     ChannelType = 13 // A boolean value.
	ChannelInt128   ChannelType = 14 // A 128-bit signed integer using github.com/shogo82148/int128.
	ChannelUint128  ChannelType = 15 // A 128-bit unsigned integer using github.com/shogo82148/int128.
	ChannelFloat128 ChannelType = 16 // A 128-bit floating point number using github.com/shogo82148/float128.
	ChannelBFloat16 ChannelType = 17 // A 16-bit brain floating point number.
)

// Returns the base channel type without the optional flags.
func (c ChannelType) Base() ChannelType {
	return c & channelTypeBaseMask
}

// Returns whether the Min value flag is set.
func (c ChannelType) HasMin() bool {
	return c&channelTypeMinFlag != 0
}

// Returns whether the Max value flag is set.
func (c ChannelType) HasMax() bool {
	return c&channelTypeMaxFlag != 0
}

// Returns a new ChannelType with the Min flag set or cleared.
func (c ChannelType) WithMin(hasMin bool) ChannelType {
	if hasMin {
		return c | channelTypeMinFlag
	}
	return c & ^channelTypeMinFlag
}

// Returns a new ChannelType with the Max flag set or cleared.
func (c ChannelType) WithMax(hasMax bool) ChannelType {
	if hasMax {
		return c | channelTypeMaxFlag
	}
	return c & ^channelTypeMaxFlag
}

// This function returns the size of each element in a channel in bytes.
func (c ChannelType) Size() int {
	switch c.Base() {
	case ChannelUnknown:
		return 0
	case ChannelInt8:
		return 1
	case ChannelInt16:
		return 2
	case ChannelInt32:
		return 4
	case ChannelInt64:
		return 8
	case ChannelUint8:
		return 1
	case ChannelUint16:
		return 2
	case ChannelUint32:
		return 4
	case ChannelUint64:
		return 8
	case ChannelFloat8:
		return 1
	case ChannelFloat16:
		return 2
	case ChannelFloat32:
		return 4
	case ChannelFloat64:
		return 8
	case ChannelBool:
		return 1
	case ChannelInt128:
		return 16
	case ChannelUint128:
		return 16
	case ChannelFloat128:
		return 16
	case ChannelBFloat16:
		return 2
	default:
		panic("pixi: unsupported channel type")
	}
}

func (c ChannelType) String() string {
	switch c.Base() {
	case ChannelUnknown:
		return "unknown"
	case ChannelInt8:
		return "int8"
	case ChannelInt16:
		return "int16"
	case ChannelInt32:
		return "int32"
	case ChannelInt64:
		return "int64"
	case ChannelUint8:
		return "uint8"
	case ChannelUint16:
		return "uint16"
	case ChannelUint32:
		return "uint32"
	case ChannelUint64:
		return "uint64"
	case ChannelFloat8:
		return "float8"
	case ChannelFloat16:
		return "float16"
	case ChannelFloat32:
		return "float32"
	case ChannelFloat64:
		return "float64"
	case ChannelBool:
		return "bool"
	case ChannelInt128:
		return "int128"
	case ChannelUint128:
		return "uint128"
	case ChannelFloat128:
		return "float128"
	case ChannelBFloat16:
		return "bfloat16"
	default:
		panic("pixi: unsupported channel type")
	}
}

// This function reads the value of a given ChannelType from the provided raw byte slice.
// The read operation is type-dependent, with each channel type having its own specific method
// for reading values. This ensures that the correct data is read and converted into the
// expected format.
func (c ChannelType) Value(raw []byte, o binary.ByteOrder) any {
	switch c.Base() {
	case ChannelUnknown:
		panic("pixi: tried to read channel with unknown size")
	case ChannelInt8:
		return int8(raw[0])
	case ChannelUint8:
		return raw[0]
	case ChannelInt16:
		return int16(o.Uint16(raw))
	case ChannelUint16:
		return o.Uint16(raw)
	case ChannelInt32:
		return int32(o.Uint32(raw))
	case ChannelUint32:
		return o.Uint32(raw)
	case ChannelInt64:
		return int64(o.Uint64(raw))
	case ChannelUint64:
		return o.Uint64(raw)
	case ChannelFloat8:
		return float8.Float8(uint8(raw[0]))
	case ChannelFloat16:
		return float16.Frombits(o.Uint16(raw))
	case ChannelFloat32:
		return math.Float32frombits(o.Uint32(raw))
	case ChannelFloat64:
		return math.Float64frombits(o.Uint64(raw))
	case ChannelBool:
		return raw[0] != 0
	case ChannelInt128:
		// Read 128-bit signed integer from bytes
		var h int64
		var l uint64
		if o == binary.BigEndian {
			h = int64(o.Uint64(raw[0:8]))
			l = o.Uint64(raw[8:16])
		} else {
			l = o.Uint64(raw[0:8])
			h = int64(o.Uint64(raw[8:16]))
		}
		return int128.Int128{H: h, L: l}
	case ChannelUint128:
		// Read 128-bit unsigned integer from bytes
		var h uint64
		var l uint64
		if o == binary.BigEndian {
			h = o.Uint64(raw[0:8])
			l = o.Uint64(raw[8:16])
		} else {
			l = o.Uint64(raw[0:8])
			h = o.Uint64(raw[8:16])
		}
		return int128.Uint128{H: h, L: l}
	case ChannelFloat128:
		// Read 128-bit floating point from bytes using float128 library
		var h uint64
		var l uint64
		if o == binary.BigEndian {
			h = o.Uint64(raw[0:8])
			l = o.Uint64(raw[8:16])
		} else {
			l = o.Uint64(raw[0:8])
			h = o.Uint64(raw[8:16])
		}
		return float128.FromBits(h, l)
	case ChannelBFloat16:
		// Read BFloat16 from bytes
		bits := o.Uint16(raw)
		return floatx.BF16Frombits(bits)
	default:
		panic("pixi: tried to read unsupported channel type")
	}
}

// Writes the given value, assumed to correspond to the ChannelType, into it's raw representation
// in bytes according to the byte order specified.
func (c ChannelType) PutValue(val any, o binary.ByteOrder, bytes []byte) {
	switch c.Base() {
	case ChannelUnknown:
		panic("pixi: tried to write channel with unknown size")
	case ChannelInt8:
		bytes[0] = byte(val.(int8))
	case ChannelUint8:
		bytes[0] = val.(uint8)
	case ChannelInt16:
		o.PutUint16(bytes, uint16(val.(int16)))
	case ChannelUint16:
		o.PutUint16(bytes, val.(uint16))
	case ChannelInt32:
		o.PutUint32(bytes, uint32(val.(int32)))
	case ChannelUint32:
		o.PutUint32(bytes, val.(uint32))
	case ChannelInt64:
		o.PutUint64(bytes, uint64(val.(int64)))
	case ChannelUint64:
		o.PutUint64(bytes, val.(uint64))
	case ChannelFloat8:
		bytes[0] = byte(val.(float8.Float8))
	case ChannelFloat16:
		o.PutUint16(bytes, val.(float16.Float16).Bits())
	case ChannelFloat32:
		o.PutUint32(bytes, math.Float32bits(val.(float32)))
	case ChannelFloat64:
		o.PutUint64(bytes, math.Float64bits(val.(float64)))
	case ChannelBool:
		if val.(bool) {
			bytes[0] = 1
		} else {
			bytes[0] = 0
		}
	case ChannelInt128:
		// Write 128-bit signed integer to bytes
		val128 := val.(int128.Int128)
		if o == binary.BigEndian {
			o.PutUint64(bytes[0:8], uint64(val128.H))
			o.PutUint64(bytes[8:16], val128.L)
		} else {
			o.PutUint64(bytes[0:8], val128.L)
			o.PutUint64(bytes[8:16], uint64(val128.H))
		}
	case ChannelUint128:
		// Write 128-bit unsigned integer to bytes
		val128 := val.(int128.Uint128)
		if o == binary.BigEndian {
			o.PutUint64(bytes[0:8], val128.H)
			o.PutUint64(bytes[8:16], val128.L)
		} else {
			o.PutUint64(bytes[0:8], val128.L)
			o.PutUint64(bytes[8:16], val128.H)
		}
	case ChannelFloat128:
		// Write 128-bit floating point to bytes
		val128 := val.(float128.Float128)
		h, l := val128.Bits()
		if o == binary.BigEndian {
			o.PutUint64(bytes[0:8], h)
			o.PutUint64(bytes[8:16], l)
		} else {
			o.PutUint64(bytes[0:8], l)
			o.PutUint64(bytes[8:16], h)
		}
	case ChannelBFloat16:
		// Write BFloat16 to bytes
		bf16 := val.(floatx.BFloat16)
		o.PutUint16(bytes, uint16(bf16))
	default:
		panic("pixi: tried to write unsupported channel type")
	}
}

// Compares two values based on the channel type. Returns -1 if a < b, 0 if a == b, 1 if a > b.
func (ctype ChannelType) CompareValues(a, b any) int {
	switch ctype.Base() {
	case ChannelInt8:
		va, vb := a.(int8), b.(int8)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case ChannelUint8:
		va, vb := a.(uint8), b.(uint8)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case ChannelInt16:
		va, vb := a.(int16), b.(int16)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case ChannelUint16:
		va, vb := a.(uint16), b.(uint16)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case ChannelInt32:
		va, vb := a.(int32), b.(int32)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case ChannelUint32:
		va, vb := a.(uint32), b.(uint32)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case ChannelInt64:
		va, vb := a.(int64), b.(int64)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case ChannelUint64:
		va, vb := a.(uint64), b.(uint64)
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
		return 0
	case ChannelFloat8:
		va, vb := float64(a.(float8.Float8)), float64(b.(float8.Float8))
		return cmp.Compare(va, vb)
	case ChannelFloat16:
		va, vb := a.(float16.Float16).Float32(), b.(float16.Float16).Float32()
		return cmp.Compare(va, vb)
	case ChannelFloat32:
		return cmp.Compare(a.(float32), b.(float32))
	case ChannelFloat64:
		return cmp.Compare(a.(float64), b.(float64))
	case ChannelBool:
		va, vb := a.(bool), b.(bool)
		if !va && vb {
			return -1
		}
		if va && !vb {
			return 1
		}
		return 0
	case ChannelInt128:
		va, vb := a.(int128.Int128), b.(int128.Int128)
		return va.Cmp(vb)
	case ChannelUint128:
		va, vb := a.(int128.Uint128), b.(int128.Uint128)
		return va.Cmp(vb)
	case ChannelFloat128:
		va, vb := a.(float128.Float128), b.(float128.Float128)
		return va.Compare(vb)
	case ChannelBFloat16:
		va, vb := a.(floatx.BFloat16), b.(floatx.BFloat16)
		vaf, vbf := va.Float32(), vb.Float32()
		return cmp.Compare(vaf, vbf)
	default:
		return 0
	}
}
