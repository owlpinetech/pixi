package pixi

import (
	"reflect"
	"testing"

	"github.com/owlpinetech/pixi/internal/buffer"
)

func TestPixiSampleSize(t *testing.T) {
	tests := []struct {
		name     string
		dataset  Layer
		wantSize int
	}{
		{
			name: "Empty dataset",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{},
				Fields:      []Field{},
			},
			wantSize: 0,
		},
		{
			name: "One field with size 1",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{},
				Fields:      []Field{{Name: "", Type: FieldInt8}},
			},
			wantSize: 1,
		},
		{
			name: "One field with size 2",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{},
				Fields:      []Field{{Name: "", Type: FieldInt16}},
			},
			wantSize: 2,
		},
		{
			name: "Multiple fields",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{},
				Fields:      []Field{{Name: "", Type: FieldInt8}, {Name: "", Type: FieldFloat32}},
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
		dataset  Layer
		wantSize int
	}{
		{
			name: "Empty dataset",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{},
				Fields:      []Field{},
			},
			wantSize: 0,
		},
		{
			name: "One dimension with size 10",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 10}},
				Fields:      []Field{},
			},
			wantSize: 10,
		},
		{
			name: "Multiple dimensions",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 2}, {Size: 3}},
				Fields:      []Field{},
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
		dataset  Layer
		wantSize int
	}{
		{
			name: "Empty dataset",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{},
				Fields:      []Field{},
			},
			wantSize: 0,
		},
		{
			name: "One dimension with size 10",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 10, TileSize: 5}},
				Fields:      []Field{},
			},
			wantSize: 5,
		},
		{
			name: "Multiple dimensions",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 2, TileSize: 2}, {Size: 3, TileSize: 3}},
				Fields:      []Field{},
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
		dataset  Layer
		wantSize int
	}{
		{
			name: "Empty dataset",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{},
				Fields:      []Field{},
			},
			wantSize: 0,
		},
		{
			name: "One dimension with size 10",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 10, TileSize: 10}},
				Fields:      []Field{{Type: FieldInt8}},
			},
			wantSize: 10,
		},
		{
			name: "Two dimensions with sizes 10 and 8",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 10, TileSize: 5}, {Size: 8, TileSize: 4}},
				Fields:      []Field{{Type: FieldInt8}},
			},
			wantSize: 4 * 5,
		},
		{
			name: "Three dimensions with sizes 4, 2, and 1",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 4, TileSize: 4}, {Size: 2, TileSize: 2}, {Size: 1, TileSize: 1}},
				Fields:      []Field{{Type: FieldInt8}},
			},
			wantSize: 8, // 4 * 2 * 1 = 8
		},
		{
			name: "Separate fields with always has first field size * tile size",
			dataset: Layer{
				Separated:   true,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 20, TileSize: 5}, {Size: 10, TileSize: 5}},
				Fields:      []Field{{Type: FieldFloat32}, {Type: FieldFloat64}},
			},
			wantSize: 4 * 5 * 5,
		},
		{
			name: "One dimension with tile size 5 and one field with size 2",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 10, TileSize: 5}},
				Fields:      []Field{{Type: FieldInt16}},
			},
			wantSize: 10,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotSize := test.dataset.DiskTileSize(0)
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
			dataSet := Layer{
				Separated:   tc.separated,
				Compression: CompressionNone,
				Dimensions:  tc.dims,
				Fields:      []Field{},
			}

			if dataSet.Tiles() != tc.want {
				t.Errorf("PixiTiles() = %d, want %d", dataSet.Tiles(), tc.want)
			}
		})
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

func TestAddBlankLayers(t *testing.T) {
	tests := []struct {
		name         string
		layers       []Layer
		expectLayers []Layer
	}{
		{
			"no layers", []Layer{}, []Layer{},
		},
		{
			"one layer, override compression",
			[]Layer{
				{
					Name:        "layer",
					Separated:   false,
					Compression: CompressionFlate,
					Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
					Fields:      []Field{{Name: "a", Type: FieldFloat32}, {Name: "b", Type: FieldInt16}},
				},
			},
			[]Layer{
				{
					Name:           "layer",
					Separated:      false,
					Compression:    CompressionNone,
					Dimensions:     []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
					Fields:         []Field{{Name: "a", Type: FieldFloat32}, {Name: "b", Type: FieldInt16}},
					TileOffsets:    []int64{151, 175, 199, 223},
					TileBytes:      []int64{24, 24, 24, 24},
					NextLayerStart: 0,
				},
			},
		},
		{
			"two same size layers",
			[]Layer{
				{
					Name:        "one",
					Separated:   false,
					Compression: CompressionFlate,
					Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
					Fields:      []Field{{Name: "a", Type: FieldFloat32}, {Name: "b", Type: FieldInt16}},
				},
				{
					Name:        "two",
					Separated:   false,
					Compression: CompressionFlate,
					Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
					Fields:      []Field{{Name: "c", Type: FieldFloat32}, {Name: "d", Type: FieldInt16}},
				},
			},
			[]Layer{
				{
					Name:           "one",
					Separated:      false,
					Compression:    CompressionNone,
					Dimensions:     []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
					Fields:         []Field{{Name: "a", Type: FieldFloat32}, {Name: "b", Type: FieldInt16}},
					TileOffsets:    []int64{149, 173, 197, 221},
					TileBytes:      []int64{24, 24, 24, 24},
					NextLayerStart: 221 + 24,
				},
				{
					Name:           "two",
					Separated:      false,
					Compression:    CompressionNone,
					Dimensions:     []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
					Fields:         []Field{{Name: "c", Type: FieldFloat32}, {Name: "d", Type: FieldInt16}},
					TileOffsets:    []int64{386, 410, 434, 458},
					TileBytes:      []int64{24, 24, 24, 24},
					NextLayerStart: 0,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := buffer.NewBuffer(10)
			pix, err := StartPixi(buf)
			if err != nil {
				t.Fatal(err)
			}

			diskLayers := make([]*Layer, 0)
			offset := FirstLayerOffset
			for _, layer := range tc.layers {
				dl, err := pix.AddBlankUncompressedLayer(buf, offset, layer)
				if err != nil {
					t.Fatal(err)
				}
				if dl == nil {
					t.Fatal("disk layer nil")
				}
				diskLayers = append(diskLayers, dl)

				if pix.LayerOffset(dl) != offset {
					t.Errorf("expected offset %d, got %v", offset, pix.LayerOffset(dl))
				}

				offset += dl.HeaderSize() + dl.DataSize()
			}

			for lind := range tc.expectLayers {
				expect := tc.expectLayers[lind]
				got := *diskLayers[lind]
				if !reflect.DeepEqual(expect, got) {
					t.Errorf("got %v, want %v", got, expect)
				}
			}
		})
	}
}
