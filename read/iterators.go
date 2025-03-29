package read

import (
	"io"
	"iter"
	"slices"

	"github.com/owlpinetech/pixi"
)

// Returns a sequence of every sample in the layer, in tile iteration order (efficient from a disk-loading
// perspective, each tile will only be loaded once exactly once it is needed). Each iteration contains the
// coordinate of the sample by each dimension, as well as every field of the sample.
func LayerContiguousTileOrder(r io.ReadSeeker, header *pixi.PixiHeader, layer *pixi.Layer) iter.Seq2[pixi.SampleCoordinate, []any] {
	if layer.Separated {
		panic("this iterator does not support files with separated fields")
	}
	return func(yield func(pixi.SampleCoordinate, []any) bool) {
		for tileInd := range layer.Dimensions.Tiles() {
			tileData := make([]byte, layer.DiskTileSize(tileInd))
			inTileOffset := 0
			err := layer.ReadTile(r, header, tileInd, tileData)
			if err != nil {
				return
			}
			for inTileInd := range layer.Dimensions.TileSamples() {
				coord := pixi.TileSelector{Tile: tileInd, InTile: inTileInd}.
					ToTileCoordinate(layer.Dimensions).
					ToSampleCoordinate(layer.Dimensions)
				comps := make([]any, len(layer.Fields))
				for fieldInd, field := range layer.Fields {
					comps[fieldInd] = field.BytesToValue(tileData[inTileOffset:], header.ByteOrder)
					inTileOffset += field.Size()
				}
				if !yield(coord, comps) {
					return
				}
			}
		}
	}
}

// An optimization of LayerContiguousTileOrder function for Pixi layers when only a single field of each sample
// is needed for iteration.
func LayerContiguousTileOrderSingleValue(r io.ReadSeeker, header *pixi.PixiHeader, layer *pixi.Layer, fieldName string) iter.Seq2[pixi.SampleCoordinate, any] {
	if layer.Separated {
		panic("this iterator does not support files with separated fields")
	}
	fieldInd := slices.IndexFunc(layer.Fields, func(f *pixi.Field) bool { return f.Name == fieldName })
	if fieldInd == -1 {
		panic("field to iterate over is not present in the given layer")
	}
	// number of bytes to skip to get the desired field in the sample
	fieldOffset := 0
	for range fieldInd {
		fieldOffset += layer.Fields[fieldInd].Size()
	}
	// number of bytes to skip after reading the desired field in the sample to get to the next
	fieldSkip := fieldOffset
	for i := fieldInd; i < len(layer.Fields); i++ {
		fieldSkip += layer.Fields[fieldInd].Size()
	}
	return func(yield func(pixi.SampleCoordinate, any) bool) {
		for tileInd := range layer.Dimensions.Tiles() {
			tileData := make([]byte, layer.DiskTileSize(tileInd))
			inTileOffset := fieldOffset
			err := layer.ReadTile(r, header, tileInd, tileData)
			if err != nil {
				return
			}
			for inTileInd := range layer.Dimensions.TileSamples() {
				coord := pixi.TileSelector{Tile: tileInd, InTile: inTileInd}.
					ToTileCoordinate(layer.Dimensions).
					ToSampleCoordinate(layer.Dimensions)
				val := layer.Fields[fieldInd].BytesToValue(tileData[inTileOffset:], header.ByteOrder)
				inTileOffset += fieldSkip
				if !yield(coord, val) {
					return
				}
			}
		}
	}
}
