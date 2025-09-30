package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/owlpinetech/pixi"
	"github.com/owlpinetech/pixi/edit"
	"github.com/owlpinetech/pixi/read"
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
	case "none":
		compression = pixi.CompressionNone
	default:
		fmt.Println("Invalid compression method. Must be one of: flate, lzw_lsb, lzw_msb, none")
		return
	}

	// open source and destination files
	srcStream, err := read.OpenFileOrHttp(*srcFileName)
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
		srcData := read.NewLayerReadCache(srcStream, srcPixi.Header, srcLayer, read.NewLfuCacheManager(4))
		err = edit.WriteContiguousTileOrderPixi(dstFile, dstPixi, srcPixi.AllTags(), edit.LayerWriter{
			Layer: dstLayer,
			IterFn: func(layer *pixi.Layer, coord pixi.SampleCoordinate) ([]any, map[string]any) {
				pixel, err := srcData.SampleAt(coord)
				if err != nil {
					fmt.Println("Failed to read sample from source Pixi file.")
					os.Exit(1)
				}
				return pixel, nil
			},
		})
		if err != nil {
			fmt.Println("Failed to write destination Pixi file.")
			return
		}
	}
}
