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

func TestChannelType_ValueFromBytes(t *testing.T) {
	tests := []struct {
		name        string
		channelType ChannelType
		value       any
	}{
		{"Int8", ChannelInt8, int8(-10)},
		{"Uint8", ChannelUint8, uint8(5)},
		{"Int16", ChannelInt16, int16(-1000)},
		{"Uint16", ChannelUint16, uint16(5000)},
		{"Int32", ChannelInt32, int32(-1234567)},
		{"Uint32", ChannelUint32, uint32(9876543)},
		{"Int64", ChannelInt64, int64(-2147483648)},
		{"Uint64", ChannelUint64, uint64(18446744073709551615)},
		{"Float8", ChannelFloat8, float8.ToFloat8(float32(12.75))},
		{"Float16", ChannelFloat16, float16.Fromfloat32(float32(123.456))},
		{"Float32", ChannelFloat32, float32(1.2345)},
		{"Float64", ChannelFloat64, float64(3.14159)},
		{"Bool_true", ChannelBool, true},
		{"Bool_false", ChannelBool, false},
		{"Int128", ChannelInt128, int128.Int128{H: -1, L: ^uint64(123456789012345 - 1)}},
		{"Uint128", ChannelUint128, int128.Uint128{H: 0, L: 123456789012345}},
		{"Float128", ChannelFloat128, float128.FromFloat64(-123.456)},
		{"BFloat16", ChannelBFloat16, floatx.BF16Fromfloat32(1.5)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw []byte
			switch tt.channelType {
			case ChannelInt8:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(int8))
				raw = buf.Bytes()
			case ChannelUint8:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(uint8))
				raw = buf.Bytes()
			case ChannelInt16:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(int16))
				raw = buf.Bytes()
			case ChannelUint16:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(uint16))
				raw = buf.Bytes()
			case ChannelInt32:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(int32))
				raw = buf.Bytes()
			case ChannelUint32:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(uint32))
				raw = buf.Bytes()
			case ChannelInt64:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(int64))
				raw = buf.Bytes()
			case ChannelUint64:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(uint64))
				raw = buf.Bytes()
			case ChannelFloat8:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(float8.Float8))
				raw = buf.Bytes()
			case ChannelFloat16:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(float16.Float16).Bits())
				raw = buf.Bytes()
			case ChannelFloat32:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(float32))
				raw = buf.Bytes()
			case ChannelFloat64:
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, tt.value.(float64))
				raw = buf.Bytes()
			case ChannelBool:
				if tt.value.(bool) {
					raw = []byte{1}
				} else {
					raw = []byte{0}
				}
			case ChannelInt128, ChannelUint128:
				if tt.channelType == ChannelInt128 {
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
			case ChannelFloat128:
				val128 := tt.value.(float128.Float128)
				h, l := val128.Bits()
				raw = make([]byte, 16)
				binary.BigEndian.PutUint64(raw[0:8], h)
				binary.BigEndian.PutUint64(raw[8:16], l)
			case ChannelBFloat16:
				bf16 := tt.value.(floatx.BFloat16)
				buf := bytes.NewBuffer(nil)
				binary.Write(buf, binary.BigEndian, uint16(bf16))
				raw = buf.Bytes()
			}
			val := tt.channelType.Value(raw, binary.BigEndian)
			if !reflect.DeepEqual(val, tt.value) {
				t.Errorf("Read() = %+v, want %+v", val, tt.value)
			}
		})
	}
}

func TestChannelType_WriteValue(t *testing.T) {
	tests := []struct {
		channelType  ChannelType
		writeData    []byte
		readExpected any
	}{
		{ChannelInt8, []byte{0x80}, int8(-128)},
		{ChannelUint8, []byte{0xff}, uint8(255)},
		{ChannelInt16, []byte{0xff, 0x80}, int16(-128)},
		{ChannelUint16, []byte{0xff, 0xff}, uint16(65535)},
		{ChannelInt32, []byte{0x80, 0x00, 0x00, 0x00}, int32(-2147483648)},
		{ChannelUint32, []byte{0xff, 0xff, 0xff, 0xff}, uint32(4294967295)},
		{ChannelInt64, []byte{0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, int64(-9223372036854775808)},
		{ChannelUint64, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, uint64(18446744073709551615)},
		{ChannelFloat8, []byte{0x6f}, float8.ToFloat8(float32(127.0))},
		{ChannelFloat16, []byte{0xfb, 0xff}, float16.Fromfloat32(float32(-65504.0))},
		{ChannelFloat32, []byte{0xbf, 0x80, 0x00, 0x00}, float32(-1.0)},
		{ChannelFloat64, []byte{0xbf, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, float64(-1.0)},
		{ChannelBool, []byte{0x01}, true},
		{ChannelBool, []byte{0x00}, false},
		{ChannelInt128, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfb, 0x2e}, int128.Int128{H: -1, L: ^uint64(1234 - 1)}},
		{ChannelUint128, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0xd2}, int128.Uint128{H: 0, L: 1234}},
		{ChannelFloat128, []byte{0x40, 0x09, 0x34, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, float128.FromFloat64(1234.0)},
		{ChannelBFloat16, []byte{0x3e, 0x00}, floatx.BF16Fromfloat32(1.5)},
	}

	for i, test := range tests {
		buf := make([]byte, test.channelType.Size())
		test.channelType.PutValue(test.readExpected, binary.BigEndian, buf)

		written := buf
		for b := range test.writeData {
			if test.writeData[b] != written[b] {
				t.Errorf("Test %d: unexpected write byte %d, expected %v, got %v", i+1, b, test.writeData[b], written[b])
			}
		}
	}
}

func TestChannelWriteRead(t *testing.T) {
	headers := allHeaderVariants(Version)

	cases := []Channel{
		{Name: "nameone", Type: ChannelInt8},
		{Name: "", Type: ChannelFloat64},
		{Name: "amuchlongernamethanusualwithlotsofcharacters", Type: ChannelInt16},
		{Name: "bool_channel", Type: ChannelBool},
		{Name: "int128_channel", Type: ChannelInt128},
		{Name: "uint128_channel", Type: ChannelUint128},
		{Name: "float128_channel", Type: ChannelFloat128},
		{Name: "bfloat16_channel", Type: ChannelBFloat16},
	}

	for _, c := range cases {
		for _, h := range headers {
			buf := buffer.NewBuffer(10)
			err := c.Write(buf, h)
			if err != nil {
				t.Fatal("write channel", err)
			}

			readBuf := buffer.NewBufferFrom(buf.Bytes())
			readChannel := Channel{}
			err = (&readChannel).Read(readBuf, h)
			if err != nil {
				t.Fatal("read channel", err)
			}

			if !reflect.DeepEqual(c, readChannel) {
				t.Errorf("expected read channel to be %v, got %v for header %v", c, readChannel, h)
			}
		}
	}
}

func TestChannelWithMinMaxWriteRead(t *testing.T) {
	headers := allHeaderVariants(Version)

	cases := []Channel{
		{Name: "int8_with_min", Type: ChannelInt8, Min: int8(-100), Max: nil},
		{Name: "int8_with_max", Type: ChannelInt8, Min: nil, Max: int8(100)},
		{Name: "int8_with_both", Type: ChannelInt8, Min: int8(-50), Max: int8(50)},
		{Name: "float32_with_min", Type: ChannelFloat32, Min: float32(-1.5), Max: nil},
		{Name: "float32_with_max", Type: ChannelFloat32, Min: nil, Max: float32(1.5)},
		{Name: "float32_with_both", Type: ChannelFloat32, Min: float32(-2.5), Max: float32(2.5)},
		{Name: "uint16_with_both", Type: ChannelUint16, Min: uint16(100), Max: uint16(65535)},
		{Name: "bool_with_both", Type: ChannelBool, Min: false, Max: true},
		{Name: "int64_with_min", Type: ChannelInt64, Min: int64(-9223372036854775808), Max: nil},
		{Name: "float64_with_max", Type: ChannelFloat64, Min: nil, Max: float64(3.141592653589793)},
		{Name: "int128_with_both", Type: ChannelInt128, Min: int128.Int128{H: -1, L: ^uint64(999999 - 1)}, Max: int128.Int128{H: 0, L: 999999}},
		{Name: "uint128_with_max", Type: ChannelUint128, Min: nil, Max: int128.Uint128{H: 0, L: 999999}},
		{Name: "float128_with_min", Type: ChannelFloat128, Min: float128.FromFloat64(-123.456), Max: nil},
		{Name: "bfloat16_with_both", Type: ChannelBFloat16, Min: floatx.BF16Fromfloat32(-1.0), Max: floatx.BF16Fromfloat32(1.0)},
	}

	for _, c := range cases {
		for _, h := range headers {
			buf := buffer.NewBuffer(100)
			err := c.Write(buf, h)
			if err != nil {
				t.Fatal("write channel", err)
			}

			readBuf := buffer.NewBufferFrom(buf.Bytes())
			readChannel := Channel{}
			err = (&readChannel).Read(readBuf, h)
			if err != nil {
				t.Fatal("read channel", err)
			}

			if !reflect.DeepEqual(c, readChannel) {
				t.Errorf("expected read channel to be %v, got %v for header %v", c, readChannel, h)
			}
		}
	}
}

func TestChannelTypeFlags(t *testing.T) {
	tests := []struct {
		name     string
		baseType ChannelType
		hasMin   bool
		hasMax   bool
	}{
		{"Int8 no flags", ChannelInt8, false, false},
		{"Int8 min only", ChannelInt8, true, false},
		{"Int8 max only", ChannelInt8, false, true},
		{"Int8 both flags", ChannelInt8, true, true},
		{"Float32 no flags", ChannelFloat32, false, false},
		{"Float32 min only", ChannelFloat32, true, false},
		{"Float32 max only", ChannelFloat32, false, true},
		{"Float32 both flags", ChannelFloat32, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channelType := tt.baseType.WithMin(tt.hasMin).WithMax(tt.hasMax)

			if channelType.Base() != tt.baseType {
				t.Errorf("Base() = %v, want %v", channelType.Base(), tt.baseType)
			}

			if channelType.HasMin() != tt.hasMin {
				t.Errorf("HasMin() = %v, want %v", channelType.HasMin(), tt.hasMin)
			}

			if channelType.HasMax() != tt.hasMax {
				t.Errorf("HasMax() = %v, want %v", channelType.HasMax(), tt.hasMax)
			}

			// Test that base type functionality still works
			if channelType.Size() != tt.baseType.Size() {
				t.Errorf("Size() = %v, want %v", channelType.Size(), tt.baseType.Size())
			}

			if channelType.String() != tt.baseType.String() {
				t.Errorf("String() = %v, want %v", channelType.String(), tt.baseType.String())
			}
		})
	}
}

func TestChannelUpdateMinMax(t *testing.T) {
	tests := []struct {
		name    string
		channel Channel
		values  []any
		expMin  any
		expMax  any
	}{
		{
			name:    "Int8 values",
			channel: Channel{Name: "test", Type: ChannelInt8},
			values:  []any{int8(5), int8(-10), int8(20), int8(0)},
			expMin:  int8(-10),
			expMax:  int8(20),
		},
		{
			name:    "Float32 values",
			channel: Channel{Name: "test", Type: ChannelFloat32},
			values:  []any{float32(1.5), float32(-2.5), float32(3.14), float32(0.0)},
			expMin:  float32(-2.5),
			expMax:  float32(3.14),
		},
		{
			name:    "Bool values",
			channel: Channel{Name: "test", Type: ChannelBool},
			values:  []any{false, true, false, true},
			expMin:  false,
			expMax:  true,
		},
		{
			name:    "Uint16 values",
			channel: Channel{Name: "test", Type: ChannelUint16},
			values:  []any{uint16(100), uint16(50), uint16(200), uint16(75)},
			expMin:  uint16(50),
			expMax:  uint16(200),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := tt.channel

			for _, value := range tt.values {
				channel = channel.WithMinMax(value)
			}

			if !reflect.DeepEqual(channel.Min, tt.expMin) {
				t.Errorf("UpdateMinMax() min = %v, want %v", channel.Min, tt.expMin)
			}

			if !reflect.DeepEqual(channel.Max, tt.expMax) {
				t.Errorf("UpdateMinMax() max = %v, want %v", channel.Max, tt.expMax)
			}
		})
	}
}

func TestTileOrderWriteIteratorMinMaxTracking(t *testing.T) {
	headers := allHeaderVariants(Version)

	for _, h := range headers {
		buf := buffer.NewBuffer(1000)

		// Create a simple layer with one int32 channel
		layer := NewLayer("test",
			DimensionSet{
				{Name: "x", Size: 4, TileSize: 2},
				{Name: "y", Size: 4, TileSize: 2},
			},
			ChannelSet{
				{Name: "value", Type: ChannelInt32},
			},
		)
		layer.TileBytes = make([]int64, layer.Dimensions.Tiles())
		layer.TileOffsets = make([]int64, layer.Dimensions.Tiles())

		iterator := NewTileOrderWriteIterator(buf, h, layer)

		// Set some test values
		testValues := []int32{5, -10, 20, 0, 15, -5, 30, 1, 25, -15, 8, 12, 18, 2, 22, -3}

		for _, value := range testValues {
			if !iterator.Next() {
				break
			}
			iterator.SetChannel(0, value)
		}

		iterator.Done()

		// Check that Min/Max were tracked correctly
		if layer.Channels[0].Min == nil || layer.Channels[0].Min.(int32) != -15 {
			t.Errorf("Expected min to be -15, got %v", layer.Channels[0].Min)
		}

		if layer.Channels[0].Max == nil || layer.Channels[0].Max.(int32) != 30 {
			t.Errorf("Expected max to be 30, got %v", layer.Channels[0].Max)
		}
	}
}
