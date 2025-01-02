package edit

import (
	"bytes"
	"io"

	"github.com/owlpinetech/pixi"
)

type LayerWriter struct {
	Layer  *pixi.Layer
	IterFn func(*pixi.Layer, pixi.SampleCoordinate) ([]any, map[string]any)
}

func WriteContiguousTileOrderPixi(w io.WriteSeeker, header pixi.PixiHeader, tags map[string]string, layerWriters ...LayerWriter) error {
	// write the header first
	err := header.WriteHeader(w)
	if err != nil {
		return err
	}

	// write out the tags, 0 for next start means no further sections
	tagsOffset, err := w.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	tagSection := pixi.TagSection{Tags: tags, NextTagsStart: 0}
	err = tagSection.Write(w, header)
	if err != nil {
		return err
	}

	firstlayerOffset, err := w.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}

	// update offsets to different sections
	err = header.OverwriteOffsets(w, firstlayerOffset, tagsOffset)
	if err != nil {
		return err
	}

	// write out the layers
	layerOffset := firstlayerOffset
	for layerInd, layerWriter := range layerWriters {
		// write header, then write data
		layer := layerWriter.Layer
		err = layer.WriteHeader(w, header)
		if err != nil {
			return err
		}

		for tileInd := range layerWriter.Layer.Dimensions.Tiles() {
			tileData := make([]byte, 0, layer.Dimensions.TileSamples()*layer.SampleSize())
			tileBuf := bytes.NewBuffer(tileData)
			for inTileInd := range layerWriter.Layer.Dimensions.TileSamples() {
				sampleCoord := pixi.TileSelector{Tile: tileInd, InTile: inTileInd}.
					ToTileCoordinate(layer.Dimensions).
					ToSampleCoordinate(layer.Dimensions)
				indVals, namedVals := layerWriter.IterFn(layer, sampleCoord)
				if indVals != nil {
					for fieldInd := range layer.Fields {
						err = header.Write(tileBuf, indVals[fieldInd])
						if err != nil {
							return err
						}
					}
				} else {
					for _, field := range layer.Fields {
						err = header.Write(tileBuf, namedVals[field.Name])
						if err != nil {
							return err
						}
					}
				}
			}
			err = layer.WriteTile(w, header, tileInd, tileData)
			if err != nil {
				return err
			}
		}

		if layerInd < len(layerWriters)-1 {
			nextLayerOffset, err := w.Seek(0, io.SeekCurrent)
			if err != nil {
				return err
			}
			layer.NextLayerStart = nextLayerOffset
			err = layer.OverwriteHeader(w, header, layerOffset)
			if err != nil {
				return err
			}
			layerOffset = nextLayerOffset
		}
	}

	return nil
}
