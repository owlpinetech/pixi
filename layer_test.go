package pixi

import (
	"compress/flate"
	"errors"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/owlpinetech/pixi/internal/buffer"
)

func TestLayerHeaderWriteRead(t *testing.T) {
	testCases := []struct {
		name   string
		layers []*Layer
		err    error
	}{
		{
			name: "contig",
			layers: []*Layer{{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  DimensionSet{{Size: 4, TileSize: 4}, {Size: 4, TileSize: 2}, {Size: 3, TileSize: 3}},
				Channels:    ChannelSet{{Name: "a", Type: ChannelInt32}, {Name: "b", Type: ChannelInt64}, {Name: "hello", Type: ChannelInt16}},
				TileBytes:   []int64{100, 200},
				TileOffsets: []int64{80, 160},
			}},
			err: nil,
		},
		{
			name: "layer name",
			layers: []*Layer{{
				Name:        "hello",
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  DimensionSet{{Size: 4, TileSize: 4}, {Size: 4, TileSize: 2}, {Size: 3, TileSize: 3}},
				Channels:    ChannelSet{{Name: "a", Type: ChannelInt32}, {Name: "b", Type: ChannelInt64}, {Name: "hello", Type: ChannelInt16}},
				TileBytes:   []int64{100, 200},
				TileOffsets: []int64{70, 30},
			}},
			err: nil,
		},
		{
			name: "no names",
			layers: []*Layer{{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  DimensionSet{{Size: 4, TileSize: 4}, {Size: 4, TileSize: 2}, {Size: 3, TileSize: 3}},
				Channels:    ChannelSet{{Type: ChannelInt32}, {Type: ChannelInt64}, {Type: ChannelInt16}},
				TileBytes:   []int64{100, 200},
				TileOffsets: []int64{100, 200},
			}},
			err: nil,
		},
		{
			name: "sep",
			layers: []*Layer{{
				Separated:   true,
				Compression: CompressionFlate,
				Dimensions:  DimensionSet{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
				Channels:    ChannelSet{{Name: "a", Type: ChannelFloat64}, {Name: "hello", Type: ChannelInt16}},
				TileBytes:   []int64{100, 200, 300, 400, 500, 600, 700, 800},
				TileOffsets: []int64{100, 200, 300, 400, 500, 600, 700, 800},
			}},
			err: nil,
		},
		{
			name: "tile bytes err",
			layers: []*Layer{{
				Separated:   true,
				Compression: CompressionFlate,
				Dimensions:  DimensionSet{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
				Channels:    ChannelSet{{Name: "a", Type: ChannelFloat64}, {Name: "hello", Type: ChannelInt16}},
				TileBytes:   []int64{100, 200, 300, 400, 500, 600, 700},
				TileOffsets: []int64{100, 200, 300, 400, 500, 600, 700, 800},
			}},
			err: ErrFormat("TileBytes must have same number of tiles as data set for valid pixi files"),
		},
		{
			name: "tile offsets err",
			layers: []*Layer{{
				Separated:   true,
				Compression: CompressionFlate,
				Dimensions:  DimensionSet{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
				Channels:    ChannelSet{{Name: "a", Type: ChannelFloat64}, {Name: "hello", Type: ChannelInt16}},
				TileBytes:   []int64{100, 200, 300, 400, 500, 600, 700, 800},
				TileOffsets: []int64{100, 200, 300, 400, 500, 600, 700},
			}},
			err: ErrFormat("TileOffsets must have same number of tiles as data set for valid pixi files"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// do this test with all header types
			headers := allHeaderVariants(Version)
			for _, h := range headers {
				buf := buffer.NewBuffer(10)
				err := h.WriteHeader(buf)
				if err != nil {
					t.Fatal(err)
				}
				err = tc.layers[0].WriteHeader(buf, h)
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
				readHdr := &Header{}
				err = readHdr.ReadHeader(readBuf)
				if err != nil {
					t.Fatal("read header", err)
				}

				readLayers := []*Layer{}
				for range tc.layers {
					readLayer := Layer{}
					err = (&readLayer).ReadLayer(readBuf, readHdr)
					if err != nil {
						t.Fatal("read layer", err)
					}
					readLayers = append(readLayers, &readLayer)
				}

				if !reflect.DeepEqual(tc.layers, readLayers) {
					t.Errorf("expected read dataset to be %v, got %v for header %v", tc.layers, readLayers, h)
				}
			}
		})
	}
}

func TestLayerFlateCompressionTileWriteRead(t *testing.T) {
	baseCases := allHeaderVariants(Version)
	for _, pheader := range baseCases {
		for range 25 {
			// minimum layer needed to write a tile, must have compression and tile bytes/offsets slices created
			layer := &Layer{
				Compression: CompressionFlate,
				TileBytes:   make([]int64, 5),
				TileOffsets: make([]int64, 5),
			}

			chunk := make([]byte, rand.IntN(499)+1)
			for i := range len(chunk) {
				chunk[i] = byte(rand.IntN(256))
			}

			// write tile
			buf := buffer.NewBuffer(10)
			err := layer.WriteTile(buf, pheader, 0, chunk)
			if err != nil {
				t.Fatal(err)
			}

			// read tile back
			rdr := buffer.NewBufferFrom(buf.Bytes())
			rdChunk := make([]byte, len(chunk))
			err = layer.ReadTile(rdr, pheader, 0, rdChunk)
			if err != nil {
				t.Fatal(err)
			}

			if !slices.Equal(chunk, rdChunk) {
				t.Errorf("expected chunks to be equal, got %v and %v", chunk, rdChunk)
			}
		}
	}
}

func TestLayerTileWriteReadCorrupted(t *testing.T) {
	baseCases := allHeaderVariants(Version)
	for _, pheader := range baseCases {
		// minimum layer needed to write a tile, must have compression and tile bytes/offsets slices created
		layer := &Layer{
			Compression: CompressionFlate,
			TileBytes:   make([]int64, 5),
			TileOffsets: make([]int64, 5),
		}

		chunk := make([]byte, rand.IntN(499)+1)
		for i := range len(chunk) {
			chunk[i] = byte(rand.IntN(256))
		}

		// write tile
		buf := buffer.NewBuffer(10)
		err := layer.WriteTile(buf, pheader, 0, chunk)
		if err != nil {
			t.Fatal(err)
		}

		// corrupt a byte in the data
		corruptInd := rand.IntN(len(buf.Bytes()))
		prevByte := buf.Bytes()[corruptInd]
		corruptByte := byte(rand.IntN(256))
		for corruptByte == prevByte {
			corruptByte = byte(rand.IntN(256))
		}
		buf.Bytes()[corruptInd] = corruptByte

		// read tile back
		rdr := buffer.NewBufferFrom(buf.Bytes())
		rdChunk := make([]byte, len(chunk))
		err = layer.ReadTile(rdr, pheader, 0, rdChunk)
		if err == nil {
			t.Error("expected to have an error with a corrupted byte in the tile")
		}
		var integrityErr ErrDataIntegrity
		var corruptFlate flate.CorruptInputError
		if !errors.As(err, &integrityErr) && !errors.As(err, &corruptFlate) {
			t.Errorf("expected error to be of type ErrDataIntegrity or flate.CorruptInputError, got %T", err)
		}
	}
}

func TestLayerDiskTileSize(t *testing.T) {
	tests := []struct {
		name         string
		layer        *Layer
		tileIndex    int
		expectedSize int
	}{
		{
			name: "Empty layer",
			layer: &Layer{
				Separated:  false,
				Dimensions: DimensionSet{},
				Channels:   ChannelSet{{Name: "test", Type: ChannelInt32}},
			},
			tileIndex:    0,
			expectedSize: 0,
		},
		{
			name: "Contiguous mode, single channel",
			layer: &Layer{
				Separated:  false,
				Dimensions: DimensionSet{{Size: 10, TileSize: 4}},
				Channels:   ChannelSet{{Name: "data", Type: ChannelInt32}},
			},
			tileIndex:    0,
			expectedSize: 4 * 4, // 4 samples * 4 bytes per int32
		},
		{
			name: "Contiguous mode, multiple channels",
			layer: &Layer{
				Separated:  false,
				Dimensions: DimensionSet{{Size: 8, TileSize: 4}},
				Channels:   ChannelSet{{Name: "a", Type: ChannelInt16}, {Name: "b", Type: ChannelFloat32}},
			},
			tileIndex:    0,
			expectedSize: 4 * (2 + 4), // 4 samples * (2 bytes + 4 bytes)
		},
		{
			name: "Contiguous mode, with boolean channel",
			layer: &Layer{
				Separated:  false,
				Dimensions: DimensionSet{{Size: 6, TileSize: 3}},
				Channels:   ChannelSet{{Name: "flag", Type: ChannelBool}, {Name: "value", Type: ChannelInt32}},
			},
			tileIndex:    0,
			expectedSize: 3 * (1 + 4), // 3 samples * (1 byte + 4 bytes)
		},
		{
			name: "Separated mode, non-boolean channel, first channel tile",
			layer: &Layer{
				Separated:  true,
				Dimensions: DimensionSet{{Size: 12, TileSize: 4}},
				Channels:   ChannelSet{{Name: "a", Type: ChannelInt32}, {Name: "b", Type: ChannelFloat64}},
			},
			tileIndex:    0,     // First channel (int32), first tile
			expectedSize: 4 * 4, // 4 samples * 4 bytes per int32
		},
		{
			name: "Separated mode, non-boolean channel, second channel tile",
			layer: &Layer{
				Separated:  true,
				Dimensions: DimensionSet{{Size: 12, TileSize: 4}},
				Channels:   ChannelSet{{Name: "a", Type: ChannelInt32}, {Name: "b", Type: ChannelFloat64}},
			},
			tileIndex:    3,     // Second channel (float64), first tile (tiles per dimension = 3, so tile 3 is second channel)
			expectedSize: 4 * 8, // 4 samples * 8 bytes per float64
		},
		{
			name: "Separated mode, boolean channel, exact byte boundary",
			layer: &Layer{
				Separated:  true,
				Dimensions: DimensionSet{{Size: 16, TileSize: 8}},
				Channels:   ChannelSet{{Name: "flags", Type: ChannelBool}, {Name: "data", Type: ChannelInt32}},
			},
			tileIndex:    0, // Boolean channel, first tile
			expectedSize: 1, // 8 booleans = 1 byte exactly
		},
		{
			name: "Separated mode, boolean channel, partial byte",
			layer: &Layer{
				Separated:  true,
				Dimensions: DimensionSet{{Size: 20, TileSize: 5}},
				Channels:   ChannelSet{{Name: "flags", Type: ChannelBool}, {Name: "data", Type: ChannelInt32}},
			},
			tileIndex:    0, // Boolean channel, first tile
			expectedSize: 1, // 5 booleans = 1 byte (rounded up)
		},
		{
			name: "Separated mode, boolean channel, multiple bytes",
			layer: &Layer{
				Separated:  true,
				Dimensions: DimensionSet{{Size: 30, TileSize: 17}},
				Channels:   ChannelSet{{Name: "flags", Type: ChannelBool}, {Name: "data", Type: ChannelInt32}},
			},
			tileIndex:    0, // Boolean channel, first tile
			expectedSize: 3, // 17 booleans = 3 bytes (17 + 7) / 8 = 24 / 8 = 3
		},
		{
			name: "Separated mode, mixed channels, boolean tile",
			layer: &Layer{
				Separated:  true,
				Dimensions: DimensionSet{{Size: 20, TileSize: 10}},
				Channels:   ChannelSet{{Name: "flags", Type: ChannelBool}, {Name: "count", Type: ChannelInt32}, {Name: "value", Type: ChannelFloat32}},
			},
			tileIndex:    0, // Boolean channel tile
			expectedSize: 2, // 10 booleans = 2 bytes (10 + 7) / 8 = 17 / 8 = 2
		},
		{
			name: "Separated mode, mixed channels, int32 tile",
			layer: &Layer{
				Separated:  true,
				Dimensions: DimensionSet{{Size: 20, TileSize: 10}},
				Channels:   ChannelSet{{Name: "flags", Type: ChannelBool}, {Name: "count", Type: ChannelInt32}, {Name: "value", Type: ChannelFloat32}},
			},
			tileIndex:    2,      // Int32 channel tile (tiles per dimension = 2, so tile 2 is second channel)
			expectedSize: 10 * 4, // 10 samples * 4 bytes per int32
		},
		{
			name: "Separated mode, mixed channels, float32 tile",
			layer: &Layer{
				Separated:  true,
				Dimensions: DimensionSet{{Size: 20, TileSize: 10}},
				Channels:   ChannelSet{{Name: "flags", Type: ChannelBool}, {Name: "count", Type: ChannelInt32}, {Name: "value", Type: ChannelFloat32}},
			},
			tileIndex:    4,      // Float32 channel tile (tiles per dimension = 2, so tile 4 is third channel)
			expectedSize: 10 * 4, // 10 samples * 4 bytes per float32
		},
		{
			name: "Separated mode, multiple dimensions with boolean",
			layer: &Layer{
				Separated:  true,
				Dimensions: DimensionSet{{Size: 8, TileSize: 4}, {Size: 6, TileSize: 3}},
				Channels:   ChannelSet{{Name: "active", Type: ChannelBool}},
			},
			tileIndex:    0, // Boolean channel, first tile (4 * 3 = 12 samples)
			expectedSize: 2, // 12 booleans = 2 bytes (12 + 7) / 8 = 19 / 8 = 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualSize := tt.layer.DiskTileSize(tt.tileIndex)
			if actualSize != tt.expectedSize {
				t.Errorf("DiskTileSize(%d) = %d, want %d", tt.tileIndex, actualSize, tt.expectedSize)
			}
		})
	}
}
