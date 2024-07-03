package pixi

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func TestPixiSampleSize(t *testing.T) {
	tests := []struct {
		name     string
		dataset  Summary
		wantSize int
	}{
		{
			name: "Empty dataset",
			dataset: Summary{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{},
				Fields:      []Field{},
				TileBytes:   []int64{},
			},
			wantSize: 0,
		},
		{
			name: "One field with size 1",
			dataset: Summary{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{},
				Fields:      []Field{{Name: "", Type: FieldInt8}},
				TileBytes:   []int64{},
			},
			wantSize: 1,
		},
		{
			name: "One field with size 2",
			dataset: Summary{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{},
				Fields:      []Field{{Name: "", Type: FieldInt16}},
				TileBytes:   []int64{},
			},
			wantSize: 2,
		},
		{
			name: "Multiple fields",
			dataset: Summary{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{},
				Fields:      []Field{{Name: "", Type: FieldInt8}, {Name: "", Type: FieldFloat32}},
				TileBytes:   []int64{},
			},
			wantSize: 5, // size of int8 + size of float32
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotSize := test.dataset.SampleSize()
			if gotSize != test.wantSize {
				t.Errorf("SampleSize() = %d, want %d", gotSize, test.wantSize)
			}
		})
	}
}

func TestPixiSamples(t *testing.T) {
	tests := []struct {
		name     string
		dataset  Summary
		wantSize int
	}{
		{
			name: "Empty dataset",
			dataset: Summary{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{},
				Fields:      []Field{},
				TileBytes:   []int64{},
			},
			wantSize: 0,
		},
		{
			name: "One dimension with size 10",
			dataset: Summary{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 10}},
				Fields:      []Field{},
				TileBytes:   []int64{},
			},
			wantSize: 10,
		},
		{
			name: "Multiple dimensions",
			dataset: Summary{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 2}, {Size: 3}},
				Fields:      []Field{},
				TileBytes:   []int64{},
			},
			wantSize: 6, // 2 x 3 = 6
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotSize := test.dataset.Samples()
			if gotSize != test.wantSize {
				t.Errorf("Samples() = %d, want %d", gotSize, test.wantSize)
			}
		})
	}
}

func TestPixiTileSamples(t *testing.T) {
	tests := []struct {
		name     string
		dataset  Summary
		wantSize int
	}{
		{
			name: "Empty dataset",
			dataset: Summary{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{},
				Fields:      []Field{},
				TileBytes:   []int64{},
			},
			wantSize: 0,
		},
		{
			name: "One dimension with size 10",
			dataset: Summary{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 10, TileSize: 5}},
				Fields:      []Field{},
				TileBytes:   []int64{},
			},
			wantSize: 5,
		},
		{
			name: "Multiple dimensions",
			dataset: Summary{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 2, TileSize: 2}, {Size: 3, TileSize: 3}},
				Fields:      []Field{},
				TileBytes:   []int64{},
			},
			wantSize: 6,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotSize := test.dataset.TileSamples()
			if gotSize != test.wantSize {
				t.Errorf("Samples() = %d, want %d", gotSize, test.wantSize)
			}
		})
	}
}

func TestPixiTileSize(t *testing.T) {
	tests := []struct {
		name     string
		dataset  Summary
		wantSize int64
	}{
		{
			name: "Empty dataset",
			dataset: Summary{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{},
				Fields:      []Field{},
				TileBytes:   []int64{},
			},
			wantSize: 0,
		},
		{
			name: "One dimension with size 10",
			dataset: Summary{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 10, TileSize: 10}},
				Fields:      []Field{{Type: FieldInt8}},
				TileBytes:   []int64{},
			},
			wantSize: 10,
		},
		{
			name: "Two dimensions with sizes 10 and 8",
			dataset: Summary{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 10, TileSize: 5}, {Size: 8, TileSize: 4}},
				Fields:      []Field{{Type: FieldInt8}},
				TileBytes:   []int64{},
			},
			wantSize: 4 * 5,
		},
		{
			name: "Three dimensions with sizes 4, 2, and 1",
			dataset: Summary{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 4, TileSize: 4}, {Size: 2, TileSize: 2}, {Size: 1, TileSize: 1}},
				Fields:      []Field{{Type: FieldInt8}},
				TileBytes:   []int64{},
			},
			wantSize: 8, // 4 * 2 * 1 = 8
		},
		{
			name: "Separate fields with always has first field size * tile size",
			dataset: Summary{
				Separated:   true,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 20, TileSize: 5}, {Size: 10, TileSize: 5}},
				Fields:      []Field{{Type: FieldFloat32}, {Type: FieldFloat64}},
				TileBytes:   []int64{},
			},
			wantSize: 4 * 5 * 5,
		},
		{
			name: "One dimension with tile size 5 and one field with size 2",
			dataset: Summary{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 10, TileSize: 5}},
				Fields:      []Field{{Type: FieldInt16}},
				TileBytes:   []int64{},
			},
			wantSize: 10,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotSize := test.dataset.TileSize(0)
			if gotSize != test.wantSize {
				t.Errorf("TileSize() = %d, want %d", gotSize, test.wantSize)
			}
		})
	}
}

func TestPixiTiles(t *testing.T) {
	tests := []struct {
		name      string
		dims      []Dimension
		separated bool
		want      int
	}{
		{
			name:      "two rows of 4 tiles",
			dims:      []Dimension{{Size: 86400, TileSize: 21600}, {Size: 43200, TileSize: 21600}},
			separated: false,
			want:      8,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dataSet := Summary{
				Separated:   tc.separated,
				Compression: CompressionNone,
				Dimensions:  tc.dims,
				Fields:      []Field{},
				TileBytes:   []int64{},
			}

			if dataSet.Tiles() != tc.want {
				t.Errorf("PixiTiles() = %d, want %d", dataSet.Tiles(), tc.want)
			}
		})
	}
}

func TestFieldType_Read(t *testing.T) {
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
			val := tt.fieldType.Read(raw)
			if !reflect.DeepEqual(val, tt.value) {
				t.Errorf("Read() = %+v, want %+v", val, tt.value)
			}
		})
	}
}

func TestFieldType_Write(t *testing.T) {
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
		test.fieldType.Write(buf, test.readExpected)

		written := buf
		for b := 0; b < len(test.writeData); b++ {
			if test.writeData[b] != written[b] {
				t.Errorf("Test %d: unexpected write byte %d, expected %v, got %v", i+1, b, test.writeData[b], written[b])
			}
		}
	}
}

func TestDimensionTiles(t *testing.T) {
	tests := []struct {
		name     string
		size     int
		tileSize int
		want     int
	}{
		{"size same as tile size", 10, 10, 1},
		{"small size, small tile", 100, 10, 10},
		{"medium size, medium tile", 500, 50, 10},
		{"large size, large tile", 2000, 100, 20},
		{"zero size", 0, 10, 0},
		{"negative size", -100, 10, 0},
		{"tile not multiple", 100, 11, 10},
		{"large multiple", 86400, 21600, 4},
		{"half large multiple", 43200, 21600, 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dimension := Dimension{
				Size:     test.size,
				TileSize: test.tileSize,
			}
			got := dimension.Tiles()
			if got != test.want {
				t.Errorf("got %d, want %d", got, test.want)
			}
		})
	}
}
