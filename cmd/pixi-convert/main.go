package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"image"
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
	toOrder := toPixiFlags.String("endian", "little", "the endianness byte order (big, little) to use in the Pixi file")
	toOffsetSize := toPixiFlags.Int("offsetSize", 4, "the size in bytes of offsets in the Pixi file (4 or 8)")

	fromPixiFlags := flag.NewFlagSet("fromPixi", flag.ExitOnError)
	fromSrcFile := fromPixiFlags.String("src", "", "Pixi file to convert")
	fromDstFile := fromPixiFlags.String("dst", "", "name of the file resulting from Pixi conversion")
	fromModel := fromPixiFlags.String("model", "image", "the target model to convert the Pixi file to (image)")

	switch os.Args[1] {
	case "to":
		err := toPixiFlags.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}

		if err := otherToPixi(*toSrcFile, *toDstFile, *toTileSize, *toComp, *toOrder, *toOffsetSize); err != nil {
			fmt.Println(err)
			return
		}
	case "from":
		err := fromPixiFlags.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}

		if err := pixiToOther(*fromSrcFile, *fromDstFile, *fromModel); err != nil {
			fmt.Println(err)
			return
		}
	default:
		fmt.Printf("unknown subcommand: %s\n", os.Args[1])
		fmt.Println("available subcommands: to, from")
		return
	}
}

func otherToPixi(srcFile string, dstFile string, tileSize int, comp int, endianness string, offsetSize int) error {
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

	if offsetSize != 4 && offsetSize != 8 {
		return fmt.Errorf("invalid offset size: %d; must be 4 or 8", offsetSize)
	}

	order := binary.ByteOrder(binary.BigEndian)
	switch strings.ToLower(endianness) {
	case "big":
		order = binary.BigEndian
	case "little":
		order = binary.LittleEndian
	default:
		return fmt.Errorf("invalid endianness: %s; must be 'big' or 'little'", endianness)
	}

	options := pixi.FromImageOptions{
		Compression: compression,
		OffsetSize:  pixi.OffsetSize(offsetSize),
		ByteOrder:   order,
		XTileSize:   tileSize,
		YTileSize:   tileSize,
		Tags:        map[string]string{},
	}

	var img image.Image
	var err error
	switch strings.ToLower(path.Ext(srcFile)) {
	case ".bmp":
		img, err = bmp.Decode(srcStream)
		if err != nil {
			return err
		}

	case ".png":
		img, err = png.Decode(srcStream)
		if err != nil {
			return err
		}

	case ".jpg":
		fallthrough
	case ".jpeg":
		img, err = jpeg.Decode(srcStream)
		if err != nil {
			return err
		}

	case ".tif":
		fallthrough
	case ".tiff":
		img, err = tiff.Decode(srcStream)
		if err != nil {
			return err
		}

	default:
		return pixi.ErrUnsupported("image format not yet supported for conversion to Pixi")
	}

	pixiFile, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer pixiFile.Close()

	summary := &pixi.Pixi{
		Header: pixi.NewHeader(order, pixi.OffsetSize(offsetSize)),
	}
	if err := summary.Header.WriteHeader(pixiFile); err != nil {
		return err
	}

	return summary.AppendImage(pixiFile, img, options)
}

func pixiToOther(srcFile string, dstFile string, srcModel string) error {
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

	img, err := pixi.LayerAsImage(pixiStream, pixiSum, pixiSum.Layers[0], srcModel)
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
