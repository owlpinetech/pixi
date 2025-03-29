package edit

import (
	"encoding/binary"
	"math/rand"
	"testing"

	"github.com/owlpinetech/pixi"
	"github.com/owlpinetech/pixi/internal/buffer"
)

func TestWriteContiguousTileOrder(t *testing.T) {
	// two layers, two tags
	header := &pixi.PixiHeader{Version: pixi.Version, OffsetSize: 4, ByteOrder: binary.LittleEndian}

	// layer data sets
	layerOneXSize := 10
	layerOneYSize := 20
	layerOneLum := make([]float32, layerOneXSize*layerOneYSize)
	layerOneDepth := make([]uint16, layerOneXSize*layerOneYSize)
	for i := range layerOneXSize * layerOneYSize {
		layerOneLum[i] = rand.Float32() * 100
		layerOneDepth[i] = uint16(rand.Uint32())
	}

	layerTwoXSize := 15
	layerTwoYSize := 30
	layerTwoZSize := 5
	layerTwoR := make([]uint8, layerTwoXSize*layerTwoYSize*layerTwoZSize)
	layerTwoG := make([]uint8, layerTwoXSize*layerTwoYSize*layerTwoZSize)
	layerTwoB := make([]uint8, layerTwoXSize*layerTwoYSize*layerTwoZSize)
	for i := range layerTwoXSize * layerTwoYSize * layerTwoZSize {
		layerTwoR[i] = uint8(rand.Intn(256))
		layerTwoG[i] = uint8(rand.Intn(256))
		layerTwoB[i] = uint8(rand.Intn(256))
	}

	// actually write it out
	buf := buffer.NewBuffer(20)
	err := WriteContiguousTileOrderPixi(buf, header, map[string]string{"keyOne": "valOne", "keyTwo": "valTwoExtra"},
		LayerWriter{
			Layer: pixi.NewLayer(
				"layerOne",
				false,
				pixi.CompressionNone,
				pixi.DimensionSet{
					{Name: "x", Size: layerOneXSize, TileSize: 5},
					{Name: "y", Size: layerOneYSize, TileSize: 5}},
				[]pixi.Field{
					{Name: "lum", Type: pixi.FieldFloat32},
					{Name: "depth", Type: pixi.FieldUint16}},
			),
			IterFn: func(layer *pixi.Layer, coord pixi.SampleCoordinate) ([]any, map[string]any) {
				ind := coord.ToSampleIndex(layer.Dimensions)
				return []any{layerOneLum[ind], layerOneDepth[ind]}, nil
			},
		},
		LayerWriter{
			Layer: pixi.NewLayer(
				"layerTwo",
				false,
				pixi.CompressionFlate,
				pixi.DimensionSet{
					{Name: "x", Size: layerTwoXSize, TileSize: 5},
					{Name: "y", Size: layerTwoYSize, TileSize: 5},
					{Name: "z", Size: layerTwoZSize, TileSize: 5}},
				[]pixi.Field{
					{Name: "r", Type: pixi.FieldUint8},
					{Name: "g", Type: pixi.FieldUint8},
					{Name: "b", Type: pixi.FieldUint8}},
			),
			IterFn: func(layer *pixi.Layer, coord pixi.SampleCoordinate) ([]any, map[string]any) {
				ind := coord.ToSampleIndex(layer.Dimensions)
				return []any{layerTwoR[ind], layerTwoG[ind], layerTwoB[ind]}, nil
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	// Read back and verify metadata
	readBuf := buffer.NewBufferFrom(buf.Bytes())
	readPixi, err := pixi.ReadPixi(readBuf)
	if err != nil {
		t.Fatal(err)
	}

	// tags good
	tags := readPixi.Tags[0].Tags
	if tags["keyOne"] != "valOne" || tags["keyTwo"] != "valTwoExtra" {
		t.Errorf("did not get expected tags: %v", tags)
	}

	// layer names good
	layerOne := readPixi.Layers[0]
	layerTwo := readPixi.Layers[1]
	if layerOne.Name != "layerOne" || layerTwo.Name != "layerTwo" {
		t.Errorf("did not get expected layer names: %v, %v", layerOne.Name, layerTwo.Name)
	}

	// dimensions good
	if len(layerOne.Dimensions) != 2 || len(layerTwo.Dimensions) != 3 {
		t.Errorf("expected different dimensions for layers: %v, %v", len(layerOne.Dimensions), len(layerTwo.Dimensions))
	}

	// fields good
	if len(layerOne.Fields) != 2 || len(layerTwo.Fields) != 3 {
		t.Errorf("expected different fields for layers: %v, %v", len(layerOne.Fields), len(layerTwo.Fields))
	}

	// tiles written properly
	for i := range layerOne.TileBytes {
		if layerOne.TileBytes[i] == 0 || layerOne.TileOffsets[i] == 0 {
			t.Errorf("expected non-zero tile bytes and offsets for layer 1 tile %d: %v, %v", i, layerOne.TileBytes[i], layerOne.TileOffsets[i])
		}
	}
	for i := range layerTwo.TileBytes {
		if layerTwo.TileBytes[i] == 0 || layerTwo.TileOffsets[i] == 0 {
			t.Errorf("expected non-zero tile bytes and offsets for layer 2 tile %d: %v, %v", i, layerTwo.TileBytes[i], layerTwo.TileOffsets[i])
		}
	}
}
