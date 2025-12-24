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
				Dimensions:  DimensionSet{},
				Channels:    ChannelSet{},
			},
			wantSize: 0,
		},
		{
			name: "One dimension with size 10",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  DimensionSet{{Size: 10}},
				Channels:    ChannelSet{},
			},
			wantSize: 10,
		},
		{
			name: "Multiple dimensions",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  DimensionSet{{Size: 2}, {Size: 3}},
				Channels:    ChannelSet{},
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
				Dimensions:  DimensionSet{},
				Channels:    ChannelSet{},
			},
			wantSize: 0,
		},
		{
			name: "One dimension with size 10",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  DimensionSet{{Size: 10, TileSize: 5}},
				Channels:    ChannelSet{},
			},
			wantSize: 5,
		},
		{
			name: "Multiple dimensions",
			dataset: Layer{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  DimensionSet{{Size: 2, TileSize: 2}, {Size: 3, TileSize: 3}},
				Channels:    ChannelSet{},
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

func TestPixiTiles(t *testing.T) {
	tests := []struct {
		name      string
		dims      DimensionSet
		separated bool
		want      int
	}{
		{
			name:      "two rows of 4 tiles",
			dims:      DimensionSet{{Size: 86400, TileSize: 21600}, {Size: 43200, TileSize: 21600}},
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
