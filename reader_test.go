package pixi

import (
	"reflect"
	"testing"

	"github.com/owlpinetech/pixi/internal/buffer"
)

func TestWriteReadDataSet(t *testing.T) {
	testCases := []struct {
		name string
		data Pixi
		err  error
	}{
		{
			name: "contig",
			data: Pixi{
				Layers: []*DiskLayer{{
					Layer: Layer{
						Separated:   false,
						Compression: CompressionNone,
						Dimensions:  []Dimension{{Size: 4, TileSize: 4}, {Size: 4, TileSize: 2}, {Size: 3, TileSize: 3}},
						Fields:      []Field{{Name: "a", Type: FieldInt32}, {Name: "b", Type: FieldInt64}, {Name: "hello", Type: FieldInt16}},
					},
					TileBytes:   []int64{100, 200},
					TileOffsets: []int64{80, 160},
				}}},
			err: nil,
		},
		{
			name: "layer name",
			data: Pixi{
				Layers: []*DiskLayer{{
					Layer: Layer{
						Name:        "hello",
						Separated:   false,
						Compression: CompressionNone,
						Dimensions:  []Dimension{{Size: 4, TileSize: 4}, {Size: 4, TileSize: 2}, {Size: 3, TileSize: 3}},
						Fields:      []Field{{Name: "a", Type: FieldInt32}, {Name: "b", Type: FieldInt64}, {Name: "hello", Type: FieldInt16}},
					},
					TileBytes:   []int64{100, 200},
					TileOffsets: []int64{70, 30},
				}}},
			err: nil,
		},
		{
			name: "no names",
			data: Pixi{
				Layers: []*DiskLayer{{
					Layer: Layer{
						Separated:   false,
						Compression: CompressionNone,
						Dimensions:  []Dimension{{Size: 4, TileSize: 4}, {Size: 4, TileSize: 2}, {Size: 3, TileSize: 3}},
						Fields:      []Field{{Type: FieldInt32}, {Type: FieldInt64}, {Type: FieldInt16}},
					},
					TileBytes:   []int64{100, 200},
					TileOffsets: []int64{100, 200},
				}}},
			err: nil,
		},
		{
			name: "sep",
			data: Pixi{
				Layers: []*DiskLayer{{
					Layer: Layer{
						Separated:   true,
						Compression: CompressionFlate,
						Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
						Fields:      []Field{{Name: "a", Type: FieldFloat64}, {Name: "hello", Type: FieldInt16}},
					},
					TileBytes:   []int64{100, 200, 300, 400, 500, 600, 700, 800},
					TileOffsets: []int64{100, 200, 300, 400, 500, 600, 700, 800},
				}}},
			err: nil,
		},
		{
			name: "tile bytes err",
			data: Pixi{
				Layers: []*DiskLayer{{
					Layer: Layer{
						Separated:   true,
						Compression: CompressionFlate,
						Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
						Fields:      []Field{{Name: "a", Type: FieldFloat64}, {Name: "hello", Type: FieldInt16}},
					},
					TileBytes:   []int64{100, 200, 300, 400, 500, 600, 700},
					TileOffsets: []int64{100, 200, 300, 400, 500, 600, 700, 800},
				}}},
			err: FormatError("TileBytes must have same number of tiles as data set for valid pixi files"),
		},
		{
			name: "tile offsets err",
			data: Pixi{
				Layers: []*DiskLayer{{
					Layer: Layer{
						Separated:   true,
						Compression: CompressionFlate,
						Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
						Fields:      []Field{{Name: "a", Type: FieldFloat64}, {Name: "hello", Type: FieldInt16}},
					},
					TileBytes:   []int64{100, 200, 300, 400, 500, 600, 700, 800},
					TileOffsets: []int64{100, 200, 300, 400, 500, 600, 700},
				}}},
			err: FormatError("TileOffsets must have same number of tiles as data set for valid pixi files"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := buffer.NewBuffer(10)
			_, err := StartPixi(buf)
			if err != nil {
				t.Fatal(err)
			}
			err = WriteLayer(buf, *tc.data.Layers[0])
			if tc.err != nil {
				if err == nil {
					t.Fatalf("expected error %v but got none", tc.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			readBuf := buffer.NewBufferFrom(buf.Bytes())
			ds, err := ReadPixi(readBuf)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(tc.data, ds) {
				t.Errorf("expected read dataset to be %v, got %v", tc.data, ds)
			}
		})
	}
}
