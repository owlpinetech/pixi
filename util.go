package gopixi

import (
	"io"
	"math/rand"
)

// Used in test packages to help generate random dimension sets for better test coverage.
func newRandomValidDimensionSet(maxDims int, maxDimSize int, maxTileSize int) DimensionSet {
	dimCount := rand.Intn(maxDims-1) + 1
	dims := make(DimensionSet, dimCount)
	for i := range dims {
		size := rand.Intn(maxDimSize) + 1
		tileSize := size / (rand.Intn(maxTileSize) + 1)
		if tileSize == 0 {
			tileSize = size
		}
		dims[i] = Dimension{Size: size, TileSize: tileSize}
	}
	return DimensionSet(dims)
}

// Creates a new blank uncompressed layer, initializing all channels and allocating space for all tiles in the data set with
// blank (zeroed) data. The backing WriteSeeker is left at the end of the written data, ready for further writes. This function
// assumes that the PixiHeader has already been written to the backing stream, and that the stream cursor is at the correct
// offset for writing the layer header. If the write fails partway through, an error is returned, but the backing stream may be
// partially written. Otherwise, returns a pointer to the created Layer, with supporting channels ready for further read/write access.
func newBlankUncompressedLayer(backing io.WriteSeeker, header Header, name string, dimensions DimensionSet, channels ChannelSet, opts ...LayerOption) (Layer, error) {
	uncompOpts := append(opts, WithCompression(CompressionNone))
	layer := NewLayer(name, dimensions, channels, uncompOpts...)
	err := layer.WriteHeader(backing, header)
	if err != nil {
		return Layer{}, err
	}

	for tileIndex := range layer.DiskTiles() {
		tileData := make([]byte, layer.DiskTileSize(tileIndex))
		err = layer.WriteTile(backing, header, tileIndex, tileData)
		if err != nil {
			return Layer{}, err
		}
	}

	return layer, nil
}
