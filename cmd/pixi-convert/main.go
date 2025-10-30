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

	"github.com/owlpinetech/pixi"
	"github.com/owlpinetech/pixi/edit"
)

// This application converts images to Pixi files, or Pixi files of a compatible structure to images. It serves
// as an example for basic reading and writing of Pixi data.

func main() {
	toPixiFlags := flag.NewFlagSet("toPixi", flag.ExitOnError)
	toSrcFile := toPixiFlags.String("src", "", "file to convert to Pixi")
	toDstFile := toPixiFlags.String("dst", "", "name of the resulting Pixi file")
	toTileSize := toPixiFlags.Int("tileSize", 0, "the size of tiles to generate in the Pixi file, if zero (default) will be the same size as the image")
	toComp := toPixiFlags.Int("compression", 0, "compression to be used for data in Pixi, 0 for none, 1 for flate")

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
		img, err := png.Decode(srcStream)
		if err != nil {
			return err
		}
		return edit.PixiFromImage(pixiFile, img, options)

	case ".jpg":
		fallthrough
	case ".jpeg":
		img, err := jpeg.Decode(srcStream)
		if err != nil {
			return err
		}
		return edit.PixiFromImage(pixiFile, img, options)
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

	img, err := edit.LayerAsImage(pixiStream, pixiSum, pixiSum.Layers[0])
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
		return pixi.ErrUnsupported("image format not yet supported for conversion from Pixi")
	}
	return nil
}
