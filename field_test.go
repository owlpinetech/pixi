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
	}

	for i, test := range tests {
		buf := make([]byte, test.fieldType.Size())
		test.fieldType.ValueToBytes(test.readExpected, binary.BigEndian, buf)

		written := buf
		for b := 0; b < len(test.writeData); b++ {
			if test.writeData[b] != written[b] {
				t.Errorf("Test %d: unexpected write byte %d, expected %v, got %v", i+1, b, test.writeData[b], written[b])
			}
		}
	}
}

func TestFieldWriteRead(t *testing.T) {
	headers := []PixiHeader{
		{Version: 1, ByteOrder: binary.BigEndian, OffsetSize: 4},
		{Version: 1, ByteOrder: binary.BigEndian, OffsetSize: 8},
		{Version: 1, ByteOrder: binary.LittleEndian, OffsetSize: 4},
		{Version: 1, ByteOrder: binary.LittleEndian, OffsetSize: 8},
	}

	cases := []Field{
		{Name: "nameone", Type: FieldInt8},
		{Name: "", Type: FieldFloat64},
		{Name: "amuchlongernamethanusualwithlotsofcharacters", Type: FieldInt16},
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
