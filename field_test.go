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
		{Name: "nameone", Type: FieldInt8, Max: int8(100), Min: int8(-100)},
		{Name: "", Type: FieldFloat64, Max: float64(3.14159), Min: float64(-3.14159)},
		{Name: "amuchlongernamethanusualwithlotsofcharacters", Type: FieldInt16, Max: int16(32767), Min: int16(-32768)},
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

func TestFieldMaxMin(t *testing.T) {
	headers := allHeaderVariants(Version)

	tests := []struct {
		name  string
		field Field
	}{
		{
			name: "Int8 field with max/min",
			field: Field{
				Name: "temperature",
				Type: FieldInt8,
				Max:  int8(127),
				Min:  int8(-128),
			},
		},
		{
			name: "Uint32 field with max/min",
			field: Field{
				Name: "count",
				Type: FieldUint32,
				Max:  uint32(4294967295),
				Min:  uint32(0),
			},
		},
		{
			name: "Float32 field with max/min",
			field: Field{
				Name: "pressure",
				Type: FieldFloat32,
				Max:  float32(1013.25),
				Min:  float32(950.0),
			},
		},
		{
			name: "Float64 field with max/min",
			field: Field{
				Name: "elevation",
				Type: FieldFloat64,
				Max:  float64(8848.86),
				Min:  float64(-10994.0),
			},
		},
		{
			name: "Field without max/min",
			field: Field{
				Name: "basic",
				Type: FieldInt32,
				Max:  nil,
				Min:  nil,
			},
		},
		{
			name: "Field with only max",
			field: Field{
				Name: "positive_only",
				Type: FieldUint16,
				Max:  uint16(1000),
				Min:  nil,
			},
		},
		{
			name: "Field with only min",
			field: Field{
				Name: "threshold",
				Type: FieldFloat32,
				Max:  nil,
				Min:  float32(-273.15),
			},
		},
		{
			name: "Field with zero values",
			field: Field{
				Name: "centered",
				Type: FieldInt32,
				Max:  int32(0),
				Min:  int32(0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, h := range headers {
				// Test write and read
				buf := buffer.NewBuffer(100)
				err := tt.field.Write(buf, h)
				if err != nil {
					t.Fatalf("failed to write field: %v", err)
				}

				readBuf := buffer.NewBufferFrom(buf.Bytes())
				var readField Field
				err = readField.Read(readBuf, h)
				if err != nil {
					t.Fatalf("failed to read field: %v", err)
				}

				// Verify all fields match
				if readField.Name != tt.field.Name {
					t.Errorf("name mismatch: expected %s, got %s", tt.field.Name, readField.Name)
				}
				if readField.Type != tt.field.Type {
					t.Errorf("type mismatch: expected %v, got %v", tt.field.Type, readField.Type)
				}
				if !reflect.DeepEqual(readField.Max, tt.field.Max) {
					t.Errorf("max mismatch: expected %v, got %v", tt.field.Max, readField.Max)
				}
				if !reflect.DeepEqual(readField.Min, tt.field.Min) {
					t.Errorf("min mismatch: expected %v, got %v", tt.field.Min, readField.Min)
				}

				// Test header size calculation
				expectedSize := 2 + len([]byte(tt.field.Name)) + 4 // name + field type
				if tt.field.Max != nil {
					expectedSize += tt.field.Type.Size()
				}
				if tt.field.Min != nil {
					expectedSize += tt.field.Type.Size()
				}
				if tt.field.HeaderSize(h) != expectedSize {
					t.Errorf("header size mismatch: expected %d, got %d", expectedSize, tt.field.HeaderSize(h))
				}
			}
		})
	}
}

func TestFieldTypeFlagBits(t *testing.T) {
	baseType := FieldFloat32

	// Test no flags
	if baseType.hasMax() || baseType.hasMin() {
		t.Errorf("base type should not have flags set")
	}

	// Test max flag
	maxType := baseType | fieldTypeHasMax
	if !maxType.hasMax() || maxType.hasMin() {
		t.Errorf("max flag not working correctly")
	}
	if maxType.baseType() != baseType {
		t.Errorf("base type extraction failed with max flag")
	}

	// Test min flag
	minType := baseType | fieldTypeHasMin
	if minType.hasMax() || !minType.hasMin() {
		t.Errorf("min flag not working correctly")
	}
	if minType.baseType() != baseType {
		t.Errorf("base type extraction failed with min flag")
	}

	// Test both flags
	bothType := baseType | fieldTypeHasMax | fieldTypeHasMin
	if !bothType.hasMax() || !bothType.hasMin() {
		t.Errorf("both flags not working correctly")
	}
	if bothType.baseType() != baseType {
		t.Errorf("base type extraction failed with both flags")
	}
}
