package edit

import (
	"encoding/binary"
	"image"
	"image/color"
	"io"

	"github.com/owlpinetech/pixi"
	"github.com/owlpinetech/pixi/read"
)

type FromImageOptions struct {
	Compression pixi.Compression
	ByteOrder   binary.ByteOrder
	XTileSize   int
	YTileSize   int
	Tags        map[string]string
}

func PixiFromImage(w io.WriteSeeker, img image.Image, options FromImageOptions) error {
	header := &pixi.PixiHeader{Version: pixi.Version, OffsetSize: 4, ByteOrder: options.ByteOrder}
	// write the header first
	err := header.WriteHeader(w)
	if err != nil {
		return err
	}

	layer, err := ImageToLayer(img, "image", false, options.Compression, options.XTileSize, options.YTileSize)
	if err != nil {
		return err
	}

	switch img.ColorModel() {
	case color.NRGBAModel:
		options.Tags["color-model"] = "nrgba"
	case color.NRGBA64Model:
		options.Tags["color-model"] = "nrgba64"
	case color.RGBAModel:
		options.Tags["color-model"] = "rgba"
	case color.RGBA64Model:
		options.Tags["color-model"] = "rgba64"
	case color.CMYKModel:
		options.Tags["color-model"] = "cmyk"
	case color.YCbCrModel:
		options.Tags["color-model"] = "YCbCr"
	}

	return WriteContiguousTileOrderPixi(w, header, options.Tags, LayerWriter{
		Layer: layer,
		IterFn: func(layer *pixi.Layer, coord pixi.SampleCoordinate) ([]any, map[string]any) {
			pixel := img.At(coord[0], coord[1])
			switch img.ColorModel() {
			case color.NRGBAModel:
				col := pixel.(color.NRGBA)
				return []any{col.R, col.G, col.B, col.A}, nil
			case color.NRGBA64Model:
				col := pixel.(color.NRGBA64)
				return []any{col.R, col.G, col.B, col.A}, nil
			case color.RGBAModel:
				col := pixel.(color.RGBA)
				return []any{col.R, col.G, col.B, col.A}, nil
			case color.RGBA64Model:
				col := pixel.(color.RGBA64)
				return []any{col.R, col.G, col.B, col.A}, nil
			case color.CMYKModel:
				col := pixel.(color.CMYK)
				return []any{col.C, col.M, col.Y, col.K}, nil
			case color.YCbCrModel:
				col := pixel.(color.YCbCr)
				return []any{col.Y, col.Cb, col.Cr}, nil
			}
			panic("unsupported color model")
		},
	})
}

func ImageToLayer(img image.Image, layerName string, separated bool, compression pixi.Compression, xTileSize int, yTileSize int) (*pixi.Layer, error) {
	var fields pixi.FieldSet
	switch img.ColorModel() {
	case color.NRGBAModel:
		fields = pixi.FieldSet{
			{Name: "r", Type: pixi.FieldUint8},
			{Name: "g", Type: pixi.FieldUint8},
			{Name: "b", Type: pixi.FieldUint8},
			{Name: "a", Type: pixi.FieldUint8},
		}
	case color.NRGBA64Model:
		fields = pixi.FieldSet{
			{Name: "r", Type: pixi.FieldUint16},
			{Name: "g", Type: pixi.FieldUint16},
			{Name: "b", Type: pixi.FieldUint16},
			{Name: "a", Type: pixi.FieldUint16},
		}
	case color.RGBAModel:
		fields = pixi.FieldSet{
			{Name: "r", Type: pixi.FieldUint8},
			{Name: "g", Type: pixi.FieldUint8},
			{Name: "b", Type: pixi.FieldUint8},
			{Name: "a", Type: pixi.FieldUint8},
		}
	case color.RGBA64Model:
		fields = pixi.FieldSet{
			{Name: "r", Type: pixi.FieldUint16},
			{Name: "g", Type: pixi.FieldUint16},
			{Name: "b", Type: pixi.FieldUint16},
			{Name: "a", Type: pixi.FieldUint16},
		}
	case color.CMYKModel:
		fields = pixi.FieldSet{
			{Name: "c", Type: pixi.FieldUint8},
			{Name: "m", Type: pixi.FieldUint8},
			{Name: "y", Type: pixi.FieldUint8},
			{Name: "k", Type: pixi.FieldUint8},
		}
	case color.YCbCrModel:
		fields = pixi.FieldSet{
			{Name: "Y", Type: pixi.FieldUint8},
			{Name: "Cb", Type: pixi.FieldUint8},
			{Name: "Cr", Type: pixi.FieldUint8},
		}
	default:
		return nil, pixi.UnsupportedError("color model of the image not yet supported for conversion to Pixi")
	}

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	if xTileSize == 0 {
		xTileSize = width
	}
	xTileSize = min(width, xTileSize)
	if yTileSize == 0 {
		yTileSize = height
	}
	yTileSize = min(height, yTileSize)

	return pixi.NewLayer(
		layerName,
		separated,
		compression,
		pixi.DimensionSet{
			{Name: "x", Size: width, TileSize: xTileSize},
			{Name: "y", Size: height, TileSize: yTileSize}},
		fields), nil
}

func LayerAsImage(r io.ReadSeeker, pixImg *pixi.Pixi, layer *pixi.Layer) (image.Image, error) {
	width := layer.Dimensions[0].Size
	height := layer.Dimensions[1].Size

	switch pixImg.Tags[0].Tags["color-model"] {
	case "nrgba":
		nrgbaImg := image.NewNRGBA(image.Rect(0, 0, width, height))
		for coord, comps := range read.LayerContiguousTileOrder(r, pixImg.Header, layer) {
			nrgbaImg.Set(coord[0], coord[1],
				color.NRGBA{comps[0].(uint8), comps[1].(uint8), comps[2].(uint8), comps[3].(uint8)})
		}
		return nrgbaImg, nil
	case "nrgba64":
		nrgba64Img := image.NewNRGBA64(image.Rect(0, 0, width, height))
		for coord, comps := range read.LayerContiguousTileOrder(r, pixImg.Header, layer) {
			nrgba64Img.Set(coord[0], coord[1],
				color.NRGBA64{comps[0].(uint16), comps[1].(uint16), comps[2].(uint16), comps[3].(uint16)})
		}
		return nrgba64Img, nil
	case "rgba":
		rgbaImg := image.NewRGBA(image.Rect(0, 0, width, height))
		for coord, comps := range read.LayerContiguousTileOrder(r, pixImg.Header, layer) {
			rgbaImg.Set(coord[0], coord[1],
				color.RGBA{comps[0].(uint8), comps[1].(uint8), comps[2].(uint8), comps[3].(uint8)})
		}
		return rgbaImg, nil
	case "rgba64":
		rgba64Img := image.NewRGBA64(image.Rect(0, 0, width, height))
		for coord, comps := range read.LayerContiguousTileOrder(r, pixImg.Header, layer) {
			rgba64Img.Set(coord[0], coord[1],
				color.NRGBA64{comps[0].(uint16), comps[1].(uint16), comps[2].(uint16), comps[3].(uint16)})
		}
		return rgba64Img, nil
	case "cmyk":
		cmykImg := image.NewCMYK(image.Rect(0, 0, width, height))
		for coord, comps := range read.LayerContiguousTileOrder(r, pixImg.Header, layer) {
			cmykImg.Set(coord[0], coord[1],
				color.CMYK{comps[0].(uint8), comps[1].(uint8), comps[2].(uint8), comps[3].(uint8)})
		}
		return cmykImg, nil
	case "YCbCr":
		ycbcrImg := image.NewYCbCr(image.Rect(0, 0, width, height), image.YCbCrSubsampleRatio420)
		for coord, comps := range read.LayerContiguousTileOrder(r, pixImg.Header, layer) {
			yOff := ycbcrImg.YOffset(coord[0], coord[1])
			cOff := ycbcrImg.COffset(coord[0], coord[1])
			ycbcrImg.Y[yOff] = comps[0].(uint8)
			ycbcrImg.Cb[cOff] = comps[1].(uint8)
			ycbcrImg.Cr[cOff] = comps[2].(uint8)
		}
		return ycbcrImg, nil
	default:
		return nil, pixi.UnsupportedError("color model of the layer not yet supported for conversion to Pixi")
	}
}
