package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"image/jpeg"
	"image/png"
	"os"
	"path"
	"strings"

	"github.com/owlpinetech/pixi"
	"github.com/owlpinetech/pixi/edit"
)

func main() {
	toPixiFlags := flag.NewFlagSet("toPixi", flag.ExitOnError)
	toSrcFile := toPixiFlags.String("src", "", "file to convert to Pixi")
	toDstFile := toPixiFlags.String("dst", "", "name of the resulting Pixi file")
	toTileSize := toPixiFlags.Int("tileSize", 0, "the size of tiles to generate in the Pixi file, if 0 will be calculated automatically")
	toComp := toPixiFlags.Int("compression", 0, "compression to be used for data in Pixi, 0 for none, 1 for flate")
	fromPixiFlags := flag.NewFlagSet("fromPixi", flag.ExitOnError)
	fromSrcFile := fromPixiFlags.String("src", "", "Pixi file to convert")
	fromDstFile := fromPixiFlags.String("dst", "", "name of the file resulting from Pixi conversion")

	switch os.Args[1] {
	case "to":
		err := toPixiFlags.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}

		if err := otherToPixi(*toSrcFile, *toDstFile, *toTileSize, *toComp); err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
	case "from":
		err := fromPixiFlags.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}

		if err := pixiToOther(*fromSrcFile, *fromDstFile); err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
	default:
		fmt.Printf("unknown subcommand: %s\n", os.Args[1])
		fmt.Println("available subcommands: to, from")
		os.Exit(-1)
	}
}

func otherToPixi(srcFile string, dstFile string, tileSize int, comp int) error {
	rdFile, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer rdFile.Close()

	pixiFile, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer pixiFile.Close()

	compression := pixi.CompressionNone
	if comp == 1 {
		compression = pixi.CompressionFlate
	}
	options := edit.FromImageOptions{
		Compression: compression,
		ByteOrder:   binary.BigEndian,
		XTileSize:   tileSize,
		YTileSize:   tileSize,
		Tags:        map[string]string{},
	}

	switch strings.ToLower(path.Ext(srcFile)) {
	case ".png":
		img, err := png.Decode(rdFile)
		if err != nil {
			return err
		}
		return edit.PixiFromImage(pixiFile, img, options)

	case ".jpg":
		fallthrough
	case ".jpeg":
		img, err := jpeg.Decode(rdFile)
		if err != nil {
			return err
		}
		return edit.PixiFromImage(pixiFile, img, options)
	}

	return pixi.UnsupportedError("image format not yet supported for conversion to Pixi")
}

func pixiToOther(srcFile string, dstFile string) error {
	pixiFile, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer pixiFile.Close()

	imgFile, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer imgFile.Close()

	pixiSum, err := pixi.ReadPixi(pixiFile)
	if err != nil {
		return err
	}

	img, err := edit.LayerAsImage(pixiFile, &pixiSum, pixiSum.Layers[0])
	if err != nil {
		return err
	}

	switch strings.ToLower(path.Ext(dstFile)) {
	case ".png":
		err = png.Encode(imgFile, img)
		if err != nil {
			return err
		}
	case ".jpg":
		fallthrough
	case ".jpeg":
		err = jpeg.Encode(imgFile, img, nil)
		if err != nil {
			return err
		}
	default:
		return pixi.UnsupportedError("image format not yet supported for conversion to Pixi")
	}
	return nil
}
