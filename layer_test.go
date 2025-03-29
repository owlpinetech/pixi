package pixi

import (
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
				Dimensions:  []Dimension{{Size: 4, TileSize: 4}, {Size: 4, TileSize: 2}, {Size: 3, TileSize: 3}},
				Fields:      []Field{{Name: "a", Type: FieldInt32}, {Name: "b", Type: FieldInt64}, {Name: "hello", Type: FieldInt16}},
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
				Dimensions:  []Dimension{{Size: 4, TileSize: 4}, {Size: 4, TileSize: 2}, {Size: 3, TileSize: 3}},
				Fields:      []Field{{Name: "a", Type: FieldInt32}, {Name: "b", Type: FieldInt64}, {Name: "hello", Type: FieldInt16}},
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
				Dimensions:  []Dimension{{Size: 4, TileSize: 4}, {Size: 4, TileSize: 2}, {Size: 3, TileSize: 3}},
				Fields:      []Field{{Type: FieldInt32}, {Type: FieldInt64}, {Type: FieldInt16}},
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
				Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
				Fields:      []Field{{Name: "a", Type: FieldFloat64}, {Name: "hello", Type: FieldInt16}},
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
				Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
				Fields:      []Field{{Name: "a", Type: FieldFloat64}, {Name: "hello", Type: FieldInt16}},
				TileBytes:   []int64{100, 200, 300, 400, 500, 600, 700},
				TileOffsets: []int64{100, 200, 300, 400, 500, 600, 700, 800},
			}},
			err: FormatError("TileBytes must have same number of tiles as data set for valid pixi files"),
		},
		{
			name: "tile offsets err",
			layers: []*Layer{{
				Separated:   true,
				Compression: CompressionFlate,
				Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
				Fields:      []Field{{Name: "a", Type: FieldFloat64}, {Name: "hello", Type: FieldInt16}},
				TileBytes:   []int64{100, 200, 300, 400, 500, 600, 700, 800},
				TileOffsets: []int64{100, 200, 300, 400, 500, 600, 700},
			}},
			err: FormatError("TileOffsets must have same number of tiles as data set for valid pixi files"),
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
				readHdr := &PixiHeader{}
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
	}
}
