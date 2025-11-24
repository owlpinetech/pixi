package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/owlpinetech/pixi"
)

func main() {
	srcFileName := flag.String("src", "", "path to the pixi file to open")
	dstFileName := flag.String("dst", "", "name of the output pixi file")
	method := flag.String("method", "flate", "compression method to use (flate, lzw_lsb, lzw_msb, none)")
	flag.Parse()

	// determine compression method
	var compression pixi.Compression
	switch *method {
	case "flate":
		compression = pixi.CompressionFlate
	case "lzw_lsb":
		compression = pixi.CompressionLzwLsb
	case "lzw_msb":
		compression = pixi.CompressionLzwMsb
	case "rle8":
		compression = pixi.CompressionRle8
	case "none":
		compression = pixi.CompressionNone
	default:
		fmt.Println("Invalid compression method. Must be one of: flate, lzw_lsb, lzw_msb, none")
		return
	}

	// open source and destination files
	srcStream, err := pixi.OpenFileOrHttp(*srcFileName)
	if err != nil {
		fmt.Println("Failed to open source Pixi file:", err)
		return
	}
	defer srcStream.Close()

	dstFile, err := os.Create(*dstFileName)
	if err != nil {
		fmt.Println("Failed to create destination file.")
		return
	}
	defer dstFile.Close()

	// read source Pixi file and validate layer index
	srcPixi, err := pixi.ReadPixi(srcStream)
	if err != nil {
		fmt.Println("Failed to read source Pixi file.")
		return
	}

	// create destination Pixi file with compressed layers
	dstPixi := &pixi.PixiHeader{Version: pixi.Version, OffsetSize: srcPixi.Header.OffsetSize, ByteOrder: srcPixi.Header.ByteOrder}

	for _, srcLayer := range srcPixi.Layers {
		dstLayer := pixi.NewLayer(srcLayer.Name, srcLayer.Separated, compression, srcLayer.Dimensions, srcLayer.Fields)
		srcData := pixi.NewReadCachedLayer(pixi.NewLayerReadFifoCache(srcStream, srcPixi.Header, srcLayer, 4))

		err = dstPixi.WriteHeader(dstFile)
		if err != nil {
			fmt.Println("Failed to write Pixi header to destination Pixi file.")
			return
		}
		tagsOffset, err := dstFile.Seek(0, io.SeekCurrent)
		if err != nil {
			fmt.Println("Failed to seek in destination Pixi file.")
			return
		}
		tagSection := pixi.TagSection{Tags: srcPixi.AllTags(), NextTagsStart: 0}
		err = tagSection.Write(dstFile, dstPixi)
		if err != nil {
			fmt.Println("Failed to write tags to destination Pixi file.")
			return
		}

		firstlayerOffset, err := dstFile.Seek(0, io.SeekCurrent)
		if err != nil {
			fmt.Println("Failed to seek in destination Pixi file.")
			return
		}

		// update offsets to different sections
		err = dstPixi.OverwriteOffsets(dstFile, firstlayerOffset, tagsOffset)
		if err != nil {
			fmt.Println("Failed to overwrite offsets in destination Pixi file.")
			return
		}

		dstLayer.WriteHeader(dstFile, dstPixi)
		dstIterator := pixi.NewTileOrderWriteIterator(dstFile, dstPixi, dstLayer)

		for dstIterator.Next() {
			coord := dstIterator.Coordinate()
			pixel, err := srcData.SampleAt(coord)
			if err != nil {
				fmt.Println("Failed to read sample from source Pixi file.")
				return
			}
			dstIterator.SetSample(pixel)
		}

		dstIterator.Done()
		if dstIterator.Error() != nil {
			fmt.Println("Failed during tile writing iteration.")
			return
		}
	}
}
