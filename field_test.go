package pixi

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/chenxingqiang/go-floatx"
	"github.com/kshard/float8"
	"github.com/owlpinetech/pixi/internal/buffer"
	"github.com/shogo82148/float128"
	"github.com/shogo82148/int128"
	"github.com/x448/float16"
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
		{"Float8", FieldFloat8, float8.ToFloat8(float32(12.75))},
		{"Float16", FieldFloat16, float16.Fromfloat32(float32(123.456))},
		{"Float32", FieldFloat32, float32(1.2345)},
		{"Float64", FieldFloat64, float64(3.14159)},
		{"Bool_true", FieldBool, true},
		{"Bool_false", FieldBool, false},
		{"Int128", FieldInt128, int128.Int128{H: -1, L: ^uint64(123456789012345-1)}},
		{"Uint128", FieldUint128, int128.Uint128{H: 0, L: 123456789012345}},
		{"Float128", FieldFloat128, float128.FromFloat64(-123.456)},
		{"BFloat16", FieldBFloat16, floatx.BF16Fromfloat32(1.5)},
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
			case FieldFloat8:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(float8.Float8))
				raw = buf.Bytes()
			case FieldFloat16:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(float16.Float16).Bits())
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
			case FieldInt128, FieldUint128:
				if tt.fieldType == FieldInt128 {
					val128 := tt.value.(int128.Int128)
					raw = make([]byte, 16)
					binary.BigEndian.PutUint64(raw[0:8], uint64(val128.H))
					binary.BigEndian.PutUint64(raw[8:16], val128.L)
				} else {
					val128 := tt.value.(int128.Uint128)
					raw = make([]byte, 16)
					binary.BigEndian.PutUint64(raw[0:8], val128.H)
					binary.BigEndian.PutUint64(raw[8:16], val128.L)
				}
			case FieldFloat128:
				val128 := tt.value.(float128.Float128)
				h, l := val128.Bits()
				raw = make([]byte, 16)
				binary.BigEndian.PutUint64(raw[0:8], h)
				binary.BigEndian.PutUint64(raw[8:16], l)
			case FieldBFloat16:
				bf16 := tt.value.(floatx.BFloat16)
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, uint16(bf16))
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
		{FieldFloat8, []byte{0x6f}, float8.ToFloat8(float32(127.0))},
		{FieldFloat16, []byte{0xfb, 0xff}, float16.Fromfloat32(float32(-65504.0))},
		{FieldFloat32, []byte{0xbf, 0x80, 0x00, 0x00}, float32(-1.0)},
		{FieldFloat64, []byte{0xbf, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, float64(-1.0)},
		{FieldBool, []byte{0x01}, true},
		{FieldBool, []byte{0x00}, false},
		{FieldInt128, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfb, 0x2e}, int128.Int128{H: -1, L: ^uint64(1234-1)}},
		{FieldUint128, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0xd2}, int128.Uint128{H: 0, L: 1234}},
		{FieldFloat128, []byte{0x40, 0x09, 0x34, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, float128.FromFloat64(1234.0)},
		{FieldBFloat16, []byte{0x3e, 0x00}, floatx.BF16Fromfloat32(1.5)},
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
		{Name: "int128_field", Type: FieldInt128},
		{Name: "uint128_field", Type: FieldUint128},
		{Name: "float128_field", Type: FieldFloat128},
		{Name: "bfloat16_field", Type: FieldBFloat16},
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

func TestFieldWithMinMaxWriteRead(t *testing.T) {
	headers := allHeaderVariants(Version)

	cases := []Field{
		{Name: "int8_with_min", Type: FieldInt8, Min: int8(-100), Max: nil},
		{Name: "int8_with_max", Type: FieldInt8, Min: nil, Max: int8(100)},
		{Name: "int8_with_both", Type: FieldInt8, Min: int8(-50), Max: int8(50)},
		{Name: "float32_with_min", Type: FieldFloat32, Min: float32(-1.5), Max: nil},
		{Name: "float32_with_max", Type: FieldFloat32, Min: nil, Max: float32(1.5)},
		{Name: "float32_with_both", Type: FieldFloat32, Min: float32(-2.5), Max: float32(2.5)},
		{Name: "uint16_with_both", Type: FieldUint16, Min: uint16(100), Max: uint16(65535)},
		{Name: "bool_with_both", Type: FieldBool, Min: false, Max: true},
		{Name: "int64_with_min", Type: FieldInt64, Min: int64(-9223372036854775808), Max: nil},
		{Name: "float64_with_max", Type: FieldFloat64, Min: nil, Max: float64(3.141592653589793)},
		{Name: "int128_with_both", Type: FieldInt128, Min: int128.Int128{H: -1, L: ^uint64(999999-1)}, Max: int128.Int128{H: 0, L: 999999}},
		{Name: "uint128_with_max", Type: FieldUint128, Min: nil, Max: int128.Uint128{H: 0, L: 999999}},
		{Name: "float128_with_min", Type: FieldFloat128, Min: float128.FromFloat64(-123.456), Max: nil},
		{Name: "bfloat16_with_both", Type: FieldBFloat16, Min: floatx.BF16Fromfloat32(-1.0), Max: floatx.BF16Fromfloat32(1.0)},
	}

	for _, c := range cases {
		for _, h := range headers {
			buf := buffer.NewBuffer(100)
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

func TestFieldTypeFlags(t *testing.T) {
	tests := []struct {
		name     string
		baseType FieldType
		hasMin   bool
		hasMax   bool
	}{
		{"Int8 no flags", FieldInt8, false, false},
		{"Int8 min only", FieldInt8, true, false},
		{"Int8 max only", FieldInt8, false, true},
		{"Int8 both flags", FieldInt8, true, true},
		{"Float32 no flags", FieldFloat32, false, false},
		{"Float32 min only", FieldFloat32, true, false},
		{"Float32 max only", FieldFloat32, false, true},
		{"Float32 both flags", FieldFloat32, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldType := tt.baseType.WithMin(tt.hasMin).WithMax(tt.hasMax)

			if fieldType.Base() != tt.baseType {
				t.Errorf("Base() = %v, want %v", fieldType.Base(), tt.baseType)
			}

			if fieldType.HasMin() != tt.hasMin {
				t.Errorf("HasMin() = %v, want %v", fieldType.HasMin(), tt.hasMin)
			}

			if fieldType.HasMax() != tt.hasMax {
				t.Errorf("HasMax() = %v, want %v", fieldType.HasMax(), tt.hasMax)
			}

			// Test that base type functionality still works
			if fieldType.Size() != tt.baseType.Size() {
				t.Errorf("Size() = %v, want %v", fieldType.Size(), tt.baseType.Size())
			}

			if fieldType.String() != tt.baseType.String() {
				t.Errorf("String() = %v, want %v", fieldType.String(), tt.baseType.String())
			}
		})
	}
}

func TestFieldUpdateMinMax(t *testing.T) {
	tests := []struct {
		name   string
		field  Field
		values []any
		expMin any
		expMax any
	}{
		{
			name:   "Int8 values",
			field:  Field{Name: "test", Type: FieldInt8},
			values: []any{int8(5), int8(-10), int8(20), int8(0)},
			expMin: int8(-10),
			expMax: int8(20),
		},
		{
			name:   "Float32 values",
			field:  Field{Name: "test", Type: FieldFloat32},
			values: []any{float32(1.5), float32(-2.5), float32(3.14), float32(0.0)},
			expMin: float32(-2.5),
			expMax: float32(3.14),
		},
		{
			name:   "Bool values",
			field:  Field{Name: "test", Type: FieldBool},
			values: []any{false, true, false, true},
			expMin: false,
			expMax: true,
		},
		{
			name:   "Uint16 values",
			field:  Field{Name: "test", Type: FieldUint16},
			values: []any{uint16(100), uint16(50), uint16(200), uint16(75)},
			expMin: uint16(50),
			expMax: uint16(200),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := tt.field

			for _, value := range tt.values {
				field.UpdateMinMax(value)
			}

			if !reflect.DeepEqual(field.Min, tt.expMin) {
				t.Errorf("UpdateMinMax() min = %v, want %v", field.Min, tt.expMin)
			}

			if !reflect.DeepEqual(field.Max, tt.expMax) {
				t.Errorf("UpdateMinMax() max = %v, want %v", field.Max, tt.expMax)
			}
		})
	}
}

func TestTileOrderWriteIteratorMinMaxTracking(t *testing.T) {
	headers := allHeaderVariants(Version)

	for _, h := range headers {
		buf := buffer.NewBuffer(1000)

		// Create a simple layer with one int32 field
		layer := &Layer{
			Name:        "test",
			Separated:   false,
			Compression: CompressionNone,
			Dimensions: DimensionSet{
				{Name: "x", Size: 4, TileSize: 2},
				{Name: "y", Size: 4, TileSize: 2},
			},
			Fields: FieldSet{
				{Name: "value", Type: FieldInt32},
			},
		}
		layer.TileBytes = make([]int64, layer.Dimensions.Tiles())
		layer.TileOffsets = make([]int64, layer.Dimensions.Tiles())

		iterator := NewTileOrderWriteIterator(buf, h, layer)

		// Set some test values
		testValues := []int32{5, -10, 20, 0, 15, -5, 30, 1, 25, -15, 8, 12, 18, 2, 22, -3}

		for _, value := range testValues {
			if !iterator.Next() {
				break
			}
			iterator.SetField(0, value)
		}

		iterator.Done()

		// Check that Min/Max were tracked correctly
		if layer.Fields[0].Min == nil || layer.Fields[0].Min.(int32) != -15 {
			t.Errorf("Expected min to be -15, got %v", layer.Fields[0].Min)
		}

		if layer.Fields[0].Max == nil || layer.Fields[0].Max.(int32) != 30 {
			t.Errorf("Expected max to be 30, got %v", layer.Fields[0].Max)
		}
	}
}

func TestMemoryLayerMinMaxTracking(t *testing.T) {
	headers := allHeaderVariants(Version)

	for _, h := range headers {
		buf := buffer.NewBuffer(1000)

		// Create a layer with multiple fields
		layer := &Layer{
			Name:        "test",
			Separated:   false,
			Compression: CompressionNone,
			Dimensions: DimensionSet{
				{Name: "x", Size: 2, TileSize: 2},
				{Name: "y", Size: 2, TileSize: 2},
			},
			Fields: FieldSet{
				{Name: "temperature", Type: FieldFloat32},
				{Name: "count", Type: FieldInt16},
			},
		}
		layer.TileBytes = make([]int64, layer.Dimensions.Tiles())
		layer.TileOffsets = make([]int64, layer.Dimensions.Tiles())

		memLayer := NewMemoryLayer(buf, h, layer)

		// Test SetFieldAt
		testData := []struct {
			coord SampleCoordinate
			temp  float32
			count int16
		}{
			{SampleCoordinate{0, 0}, 25.5, 10},
			{SampleCoordinate{1, 0}, -5.2, 25},
			{SampleCoordinate{0, 1}, 35.8, 5},
			{SampleCoordinate{1, 1}, 15.0, 30},
		}

		for _, data := range testData {
			err := memLayer.SetFieldAt(data.coord, 0, data.temp)
			if err != nil {
				t.Fatalf("SetFieldAt failed: %v", err)
			}
			err = memLayer.SetFieldAt(data.coord, 1, data.count)
			if err != nil {
				t.Fatalf("SetFieldAt failed: %v", err)
			}
		}

		// Check Min/Max for temperature field
		if layer.Fields[0].Min == nil || layer.Fields[0].Min.(float32) != -5.2 {
			t.Errorf("Expected temperature min to be -5.2, got %v", layer.Fields[0].Min)
		}
		if layer.Fields[0].Max == nil || layer.Fields[0].Max.(float32) != 35.8 {
			t.Errorf("Expected temperature max to be 35.8, got %v", layer.Fields[0].Max)
		}

		// Check Min/Max for count field
		if layer.Fields[1].Min == nil || layer.Fields[1].Min.(int16) != 5 {
			t.Errorf("Expected count min to be 5, got %v", layer.Fields[1].Min)
		}
		if layer.Fields[1].Max == nil || layer.Fields[1].Max.(int16) != 30 {
			t.Errorf("Expected count max to be 30, got %v", layer.Fields[1].Max)
		}

		// Test SetSampleAt
		err := memLayer.SetSampleAt(SampleCoordinate{0, 0}, []any{float32(-10.5), int16(2)})
		if err != nil {
			t.Fatalf("SetSampleAt failed: %v", err)
		}

		// Check that Min was updated
		if layer.Fields[0].Min == nil || layer.Fields[0].Min.(float32) != -10.5 {
			t.Errorf("Expected updated temperature min to be -10.5, got %v", layer.Fields[0].Min)
		}
		if layer.Fields[1].Min == nil || layer.Fields[1].Min.(int16) != 2 {
			t.Errorf("Expected updated count min to be 2, got %v", layer.Fields[1].Min)
		}
	}
}
