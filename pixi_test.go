package pixi

import (
	"testing"
)

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
			gotSize := test.dataset.Dimensions.Samples()
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
			gotSize := test.dataset.Dimensions.TileSamples()
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
				Dimensions: tc.dims,
			}

			if dataSet.Dimensions.Tiles() != tc.want {
				t.Errorf("PixiTiles() = %d, want %d", dataSet.Dimensions.Tiles(), tc.want)
			}
		})
	}
}
