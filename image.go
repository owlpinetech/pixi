package pixi

import (
	"encoding/binary"
	"image"
	"image/color"
	"io"
)

type FromImageOptions struct {
	Compression Compression
	ByteOrder   binary.ByteOrder
	XTileSize   int
	YTileSize   int
	Tags        map[string]string
}

func PixiFromImage(w io.WriteSeeker, img image.Image, options FromImageOptions) error {
	header := &PixiHeader{Version: Version, OffsetSize: 4, ByteOrder: options.ByteOrder}
	// write the header first
	err := header.WriteHeader(w)
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

	// write out the tags, 0 for next start means no further sections
	tagsOffset, err := w.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	tagSection := TagSection{Tags: options.Tags, NextTagsStart: 0}
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

	layer, err := ImageToLayer(img, "image", false, options.Compression, options.XTileSize, options.YTileSize)
	if err != nil {
		return err
	}

	layer.WriteHeader(w, header)

	writerIterator := NewTileOrderWriteIterator(w, header, layer)

	for writerIterator.Next() {
		coord := writerIterator.Coordinate()
		pixel := img.At(coord[0], coord[1])
		switch img.ColorModel() {
		case color.NRGBAModel:
			col := pixel.(color.NRGBA)
			writerIterator.SetSample([]any{col.R, col.G, col.B, col.A})
		case color.NRGBA64Model:
			col := pixel.(color.NRGBA64)
			writerIterator.SetSample([]any{col.R, col.G, col.B, col.A})
		case color.RGBAModel:
			col := pixel.(color.RGBA)
			writerIterator.SetSample([]any{col.R, col.G, col.B, col.A})
		case color.RGBA64Model:
			col := pixel.(color.RGBA64)
			writerIterator.SetSample([]any{col.R, col.G, col.B, col.A})
		case color.CMYKModel:
			col := pixel.(color.CMYK)
			writerIterator.SetSample([]any{col.C, col.M, col.Y, col.K})
		case color.YCbCrModel:
			col := pixel.(color.YCbCr)
			writerIterator.SetSample([]any{col.Y, col.Cb, col.Cr})
		default:
			panic("unsupported color model")
		}
	}

	writerIterator.Done()
	return writerIterator.Error()
}

func ImageToLayer(img image.Image, layerName string, separated bool, compression Compression, xTileSize int, yTileSize int) (*Layer, error) {
	var fields FieldSet
	switch img.ColorModel() {
	case color.NRGBAModel:
		fields = FieldSet{
			{Name: "r", Type: FieldUint8},
			{Name: "g", Type: FieldUint8},
			{Name: "b", Type: FieldUint8},
			{Name: "a", Type: FieldUint8},
		}
	case color.NRGBA64Model:
		fields = FieldSet{
			{Name: "r", Type: FieldUint16},
			{Name: "g", Type: FieldUint16},
			{Name: "b", Type: FieldUint16},
			{Name: "a", Type: FieldUint16},
		}
	case color.RGBAModel:
		fields = FieldSet{
			{Name: "r", Type: FieldUint8},
			{Name: "g", Type: FieldUint8},
			{Name: "b", Type: FieldUint8},
			{Name: "a", Type: FieldUint8},
		}
	case color.RGBA64Model:
		fields = FieldSet{
			{Name: "r", Type: FieldUint16},
			{Name: "g", Type: FieldUint16},
			{Name: "b", Type: FieldUint16},
			{Name: "a", Type: FieldUint16},
		}
	case color.CMYKModel:
		fields = FieldSet{
			{Name: "c", Type: FieldUint8},
			{Name: "m", Type: FieldUint8},
			{Name: "y", Type: FieldUint8},
			{Name: "k", Type: FieldUint8},
		}
	case color.YCbCrModel:
		fields = FieldSet{
			{Name: "Y", Type: FieldUint8},
			{Name: "Cb", Type: FieldUint8},
			{Name: "Cr", Type: FieldUint8},
		}
	default:
		return nil, ErrUnsupported("color model of the image not yet supported for conversion to Pixi")
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

	return NewLayer(
		layerName,
		separated,
		compression,
		DimensionSet{
			{Name: "x", Size: width, TileSize: xTileSize},
			{Name: "y", Size: height, TileSize: yTileSize}},
		fields), nil
}

func LayerAsImage(r io.ReadSeeker, pixImg *Pixi, layer *Layer) (image.Image, error) {
	width := layer.Dimensions[0].Size
	height := layer.Dimensions[1].Size

	iterator := NewTileOrderReadIterator(r, pixImg.Header, layer)
	defer iterator.Done()

	switch pixImg.Tags[0].Tags["color-model"] {
	case "nrgba":
		if len(layer.Fields) < 4 {
			return nil, ErrUnsupported("layer does not have enough fields for NRGBA color model")
		}
		rIndex := layer.Fields.Index("r")
		gIndex := layer.Fields.Index("g")
		bIndex := layer.Fields.Index("b")
		aIndex := layer.Fields.Index("a")
		if rIndex == -1 || gIndex == -1 || bIndex == -1 || aIndex == -1 {
			rIndex, gIndex, bIndex, aIndex = 0, 1, 2, 3
		}
		nrgbaImg := image.NewNRGBA(image.Rect(0, 0, width, height))
		for iterator.Next() {
			if iterator.Error() != nil {
				return nil, iterator.Error()
			}
			coord := iterator.Coordinate()
			sample := iterator.Sample()
			nrgbaImg.Set(coord[0], coord[1],
				color.NRGBA{sample[rIndex].(uint8), sample[gIndex].(uint8), sample[bIndex].(uint8), sample[aIndex].(uint8)})
		}
		return nrgbaImg, nil
	case "nrgba64":
		if len(layer.Fields) < 4 {
			return nil, ErrUnsupported("layer does not have enough fields for NRGBA64 color model")
		}
		nrgba64Img := image.NewNRGBA64(image.Rect(0, 0, width, height))
		rIndex := layer.Fields.Index("r")
		gIndex := layer.Fields.Index("g")
		bIndex := layer.Fields.Index("b")
		aIndex := layer.Fields.Index("a")
		if rIndex == -1 || gIndex == -1 || bIndex == -1 || aIndex == -1 {
			rIndex, gIndex, bIndex, aIndex = 0, 1, 2, 3
		}
		for iterator.Next() {
			if iterator.Error() != nil {
				return nil, iterator.Error()
			}
			coord := iterator.Coordinate()
			sample := iterator.Sample()
			nrgba64Img.Set(coord[0], coord[1],
				color.NRGBA64{sample[rIndex].(uint16), sample[gIndex].(uint16), sample[bIndex].(uint16), sample[aIndex].(uint16)})
		}
		return nrgba64Img, nil
	case "rgba":
		if len(layer.Fields) < 4 {
			return nil, ErrUnsupported("layer does not have enough fields for RGBA color model")
		}
		rgbaImg := image.NewRGBA(image.Rect(0, 0, width, height))
		rIndex := layer.Fields.Index("r")
		gIndex := layer.Fields.Index("g")
		bIndex := layer.Fields.Index("b")
		aIndex := layer.Fields.Index("a")
		if rIndex == -1 || gIndex == -1 || bIndex == -1 || aIndex == -1 {
			rIndex, gIndex, bIndex, aIndex = 0, 1, 2, 3
		}
		for iterator.Next() {
			if iterator.Error() != nil {
				return nil, iterator.Error()
			}
			coord := iterator.Coordinate()
			sample := iterator.Sample()
			rgbaImg.Set(coord[0], coord[1],
				color.RGBA{sample[0].(uint8), sample[1].(uint8), sample[2].(uint8), sample[3].(uint8)})
		}
		return rgbaImg, nil
	case "rgba64":
		if len(layer.Fields) < 4 {
			return nil, ErrUnsupported("layer does not have enough fields for RGBA64 color model")
		}
		rgba64Img := image.NewRGBA64(image.Rect(0, 0, width, height))
		rIndex := layer.Fields.Index("r")
		gIndex := layer.Fields.Index("g")
		bIndex := layer.Fields.Index("b")
		aIndex := layer.Fields.Index("a")
		if rIndex == -1 || gIndex == -1 || bIndex == -1 || aIndex == -1 {
			rIndex, gIndex, bIndex, aIndex = 0, 1, 2, 3
		}
		for iterator.Next() {
			if iterator.Error() != nil {
				return nil, iterator.Error()
			}
			coord := iterator.Coordinate()
			sample := iterator.Sample()
			rgba64Img.Set(coord[0], coord[1],
				color.NRGBA64{sample[rIndex].(uint16), sample[bIndex].(uint16), sample[gIndex].(uint16), sample[aIndex].(uint16)})
		}
		return rgba64Img, nil
	case "cmyk":
		if len(layer.Fields) < 4 {
			return nil, ErrUnsupported("layer does not have enough fields for CMYK color model")
		}
		cmykImg := image.NewCMYK(image.Rect(0, 0, width, height))
		cIndex := layer.Fields.Index("c")
		mIndex := layer.Fields.Index("m")
		yIndex := layer.Fields.Index("y")
		kIndex := layer.Fields.Index("k")
		if cIndex == -1 || mIndex == -1 || yIndex == -1 || kIndex == -1 {
			cIndex, mIndex, yIndex, kIndex = 0, 1, 2, 3
		}
		for iterator.Next() {
			if iterator.Error() != nil {
				return nil, iterator.Error()
			}
			coord := iterator.Coordinate()
			sample := iterator.Sample()
			cmykImg.Set(coord[0], coord[1],
				color.CMYK{sample[cIndex].(uint8), sample[mIndex].(uint8), sample[yIndex].(uint8), sample[kIndex].(uint8)})
		}
		return cmykImg, nil
	case "YCbCr":
		if len(layer.Fields) < 3 {
			return nil, ErrUnsupported("layer does not have enough fields for YCbCr color model")
		}
		ycbcrImg := image.NewYCbCr(image.Rect(0, 0, width, height), image.YCbCrSubsampleRatio420)
		yIndex := layer.Fields.Index("Y")
		cbIndex := layer.Fields.Index("Cb")
		crIndex := layer.Fields.Index("Cr")
		if yIndex == -1 || cbIndex == -1 || crIndex == -1 {
			yIndex, cbIndex, crIndex = 0, 1, 2
		}
		for iterator.Next() {
			if iterator.Error() != nil {
				return nil, iterator.Error()
			}
			coord := iterator.Coordinate()
			sample := iterator.Sample()
			yOff := ycbcrImg.YOffset(coord[0], coord[1])
			cOff := ycbcrImg.COffset(coord[0], coord[1])
			ycbcrImg.Y[yOff] = sample[yIndex].(uint8)
			ycbcrImg.Cb[cOff] = sample[cbIndex].(uint8)
			ycbcrImg.Cr[cOff] = sample[crIndex].(uint8)
		}
		return ycbcrImg, nil
	default:
		return nil, ErrUnsupported("color model of the layer not yet supported for conversion to Pixi")
	}
}
