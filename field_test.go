package pixi

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/owlpinetech/pixi/internal/buffer"
)

func TestFieldType_ValueFromBytes(t *testing.T) {
	tests := []struct {
		name      string
		fieldType FieldType
		value     any
	}{
		{"Int8", FieldInt8, int8(-10)},
		{"Uint8", FieldUint8, uint8(5)},
		{"Int16", FieldInt16, int16(-1000)},
		{"Uint16", FieldUint16, uint16(5000)},
		{"Int32", FieldInt32, int32(-1234567)},
		{"Uint32", FieldUint32, uint32(9876543)},
		{"Int64", FieldInt64, int64(-2147483648)},
		{"Uint64", FieldUint64, uint64(18446744073709551615)},
		{"Float32", FieldFloat32, float32(1.2345)},
		{"Float64", FieldFloat64, float64(3.14159)},
		{"Bool_true", FieldBool, true},
		{"Bool_false", FieldBool, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw []byte
			switch tt.fieldType {
			case FieldInt8:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(int8))
				raw = buf.Bytes()
			case FieldUint8:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(uint8))
				raw = buf.Bytes()
			case FieldInt16:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(int16))
				raw = buf.Bytes()
			case FieldUint16:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(uint16))
				raw = buf.Bytes()
			case FieldInt32:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(int32))
				raw = buf.Bytes()
			case FieldUint32:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(uint32))
				raw = buf.Bytes()
			case FieldInt64:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(int64))
				raw = buf.Bytes()
			case FieldUint64:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(uint64))
				raw = buf.Bytes()
			case FieldFloat32:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(float32))
				raw = buf.Bytes()
			case FieldFloat64:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(float64))
				raw = buf.Bytes()
			case FieldBool:
				if tt.value.(bool) {
					raw = []byte{1}
				} else {
					raw = []byte{0}
				}
			}
			val := tt.fieldType.BytesToValue(raw, binary.BigEndian)
			if !reflect.DeepEqual(val, tt.value) {
				t.Errorf("Read() = %+v, want %+v", val, tt.value)
			}
		})
	}
}

func TestFieldType_WriteValue(t *testing.T) {
	tests := []struct {
		fieldType    FieldType
		writeData    []byte
		readExpected any
	}{
		{FieldInt8, []byte{0x80}, int8(-128)},
		{FieldUint8, []byte{0xff}, uint8(255)},
		{FieldInt16, []byte{0xff, 0x80}, int16(-128)},
		{FieldUint16, []byte{0xff, 0xff}, uint16(65535)},
		{FieldInt32, []byte{0x80, 0x00, 0x00, 0x00}, int32(-2147483648)},
		{FieldUint32, []byte{0xff, 0xff, 0xff, 0xff}, uint32(4294967295)},
		{FieldInt64, []byte{0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, int64(-9223372036854775808)},
		{FieldUint64, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, uint64(18446744073709551615)},
		{FieldFloat32, []byte{0xbf, 0x80, 0x00, 0x00}, float32(-1.0)},
		{FieldFloat64, []byte{0xbf, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, float64(-1.0)},
		{FieldBool, []byte{0x01}, true},
		{FieldBool, []byte{0x00}, false},
	}

	for i, test := range tests {
		buf := make([]byte, test.fieldType.Size())
		test.fieldType.ValueToBytes(test.readExpected, binary.BigEndian, buf)

		written := buf
		for b := range test.writeData {
			if test.writeData[b] != written[b] {
				t.Errorf("Test %d: unexpected write byte %d, expected %v, got %v", i+1, b, test.writeData[b], written[b])
			}
		}
	}
}

func TestFieldWriteRead(t *testing.T) {
	headers := allHeaderVariants(Version)

	cases := []Field{
		{Name: "nameone", Type: FieldInt8},
		{Name: "", Type: FieldFloat64},
		{Name: "amuchlongernamethanusualwithlotsofcharacters", Type: FieldInt16},
		{Name: "bool_field", Type: FieldBool},
	}

	for _, c := range cases {
		for _, h := range headers {
			buf := buffer.NewBuffer(10)
			err := c.Write(buf, h)
			if err != nil {
				t.Fatal("write field", err)
			}

			readBuf := buffer.NewBufferFrom(buf.Bytes())
			readField := Field{}
			err = (&readField).Read(readBuf, h)
			if err != nil {
				t.Fatal("read field", err)
			}

			if !reflect.DeepEqual(c, readField) {
				t.Errorf("expected read field to be %v, got %v for header %v", c, readField, h)
			}
		}
	}
}

func TestPackBoolsToBitfield(t *testing.T) {
	tests := []struct {
		name     string
		bools    []bool
		expected []byte
	}{
		{"empty", []bool{}, []byte{}},
		{"single_true", []bool{true}, []byte{0x01}},
		{"single_false", []bool{false}, []byte{0x00}},
		{"eight_alternating", []bool{true, false, true, false, true, false, true, false}, []byte{0x55}},
		{"eight_all_true", []bool{true, true, true, true, true, true, true, true}, []byte{0xff}},
		{"nine_mixed", []bool{true, false, true, false, true, false, true, false, true}, []byte{0x55, 0x01}},
		{"fifteen_mixed", []bool{true, false, true, false, true, false, true, false, true, false, true, false, true, false, true}, []byte{0x55, 0x55}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := PackBoolsToBitfield(test.bools)
			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf("PackBoolsToBitfield() = %v, want %v", result, test.expected)
			}
		})
	}
}

func TestUnpackBitfieldToBools(t *testing.T) {
	tests := []struct {
		name     string
		bitfield []byte
		count    int
		expected []bool
	}{
		{"empty", []byte{}, 0, []bool{}},
		{"single_true", []byte{0x01}, 1, []bool{true}},
		{"single_false", []byte{0x00}, 1, []bool{false}},
		{"eight_alternating", []byte{0x55}, 8, []bool{true, false, true, false, true, false, true, false}},
		{"eight_all_true", []byte{0xff}, 8, []bool{true, true, true, true, true, true, true, true}},
		{"nine_mixed", []byte{0x55, 0x01}, 9, []bool{true, false, true, false, true, false, true, false, true}},
		{"partial_byte", []byte{0xff}, 5, []bool{true, true, true, true, true}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := UnpackBitfieldToBools(test.bitfield, test.count)
			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf("UnpackBitfieldToBools() = %v, want %v", result, test.expected)
			}
		})
	}
}
