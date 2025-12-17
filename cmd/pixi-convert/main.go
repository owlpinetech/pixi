package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gracefulearth/image/bmp"
	"github.com/gracefulearth/image/tiff"
	"github.com/owlpinetech/pixi"
)

// This application converts images to Pixi files, or Pixi files of a compatible structure to images. It serves
// as an example for basic reading and writing of Pixi data.

func main() {
	toPixiFlags := flag.NewFlagSet("toPixi", flag.ExitOnError)
	toSrcFile := toPixiFlags.String("src", "", "file to convert to Pixi")
	toDstFile := toPixiFlags.String("dst", "", "name of the resulting Pixi file")
	toTileSize := toPixiFlags.Int("tileSize", 0, "the size of tiles to generate in the Pixi file, if zero (default) will be the same size as the image")
	toComp := toPixiFlags.Int("compression", 0, "compression to be used for data in Pixi (none, flate, lzw-lsb, lzw-msb, rle8) represented as 0, 1, 2, 3, 4 respectively")

	fromPixiFlags := flag.NewFlagSet("fromPixi", flag.ExitOnError)
	fromSrcFile := fromPixiFlags.String("src", "", "Pixi file to convert")
	fromDstFile := fromPixiFlags.String("dst", "", "name of the file resulting from Pixi conversion")

	switch os.Args[1] {
	case "to":
		err := toPixiFlags.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}

		if err := otherToPixi(*toSrcFile, *toDstFile, *toTileSize, *toComp); err != nil {
			fmt.Println(err)
			return
		}
	case "from":
		err := fromPixiFlags.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}

		if err := pixiToOther(*fromSrcFile, *fromDstFile); err != nil {
			fmt.Println(err)
			return
		}
	default:
		fmt.Printf("unknown subcommand: %s\n", os.Args[1])
		fmt.Println("available subcommands: to, from")
		return
	}
}

func otherToPixi(srcFile string, dstFile string, tileSize int, comp int) error {
	var srcStream io.Reader
	if strings.HasPrefix(srcFile, "http://") || strings.HasPrefix(srcFile, "https://") {
		resp, err := http.Get(srcFile)
		if err != nil {
			return err
		}
		defer resp.Body.Close() // Close the response body when done

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("failed to fetch file, status code: %d", resp.StatusCode)
		}
		srcStream = resp.Body
	} else {
		rdFile, err := os.Open(srcFile)
		if err != nil {
			return err
		}
		defer rdFile.Close()

		srcStream = rdFile
	}

	pixiFile, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer pixiFile.Close()

	compression := pixi.CompressionNone
	switch comp {
	case 0:
		compression = pixi.CompressionNone
	case 1:
		compression = pixi.CompressionFlate
	case 2:
		compression = pixi.CompressionLzwLsb
	case 3:
		compression = pixi.CompressionLzwMsb
	case 4:
		compression = pixi.CompressionRle8
	}
	options := pixi.FromImageOptions{
		Compression: compression,
		ByteOrder:   binary.BigEndian,
		XTileSize:   tileSize,
		YTileSize:   tileSize,
		Tags:        map[string]string{},
	}

	switch strings.ToLower(path.Ext(srcFile)) {
	case ".bmp":
		img, err := bmp.Decode(srcStream)
		if err != nil {
			return err
		}
		return pixi.PixiFromImage(pixiFile, img, options)

	case ".png":
		img, err := png.Decode(srcStream)
		if err != nil {
			return err
		}
		return pixi.PixiFromImage(pixiFile, img, options)

	case ".jpg":
		fallthrough
	case ".jpeg":
		img, err := jpeg.Decode(srcStream)
		if err != nil {
			return err
		}
		return pixi.PixiFromImage(pixiFile, img, options)

	case ".tif":
		fallthrough
	case ".tiff":
		img, err := tiff.Decode(srcStream)
		if err != nil {
			return err
		}
		return pixi.PixiFromImage(pixiFile, img, options)
	}

	return pixi.ErrUnsupported("image format not yet supported for conversion to Pixi")
}

func pixiToOther(srcFile string, dstFile string) error {
	pixiStream, err := pixi.OpenFileOrHttp(srcFile)
	if err != nil {
		return err
	}
	defer pixiStream.Close()

	imgFile, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer imgFile.Close()

	pixiSum, err := pixi.ReadPixi(pixiStream)
	if err != nil {
		return err
	}

	img, err := pixi.LayerAsImage(pixiStream, pixiSum, pixiSum.Layers[0])
	if err != nil {
		return err
	}

	switch strings.ToLower(path.Ext(dstFile)) {
	case ".bmp":
		err = bmp.Encode(imgFile, img)
		if err != nil {
			return err
		}
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
	case ".tif":
		fallthrough
	case ".tiff":
		err = tiff.Encode(imgFile, img, nil)
		if err != nil {
			return err
		}
	default:
		return pixi.ErrUnsupported("image format not yet supported for conversion from Pixi")
	}
	return nil
}
