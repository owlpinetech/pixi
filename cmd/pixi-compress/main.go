package main

import (
	"flag"
	"fmt"
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

	// read source Pixi file and validate layer index
	srcPixi, err := pixi.ReadPixi(srcStream)
	if err != nil {
		fmt.Println("Failed to read source Pixi file.")
		return
	}

	dstFile, err := os.Create(*dstFileName)
	if err != nil {
		fmt.Println("Failed to create destination file.")
		return
	}
	defer dstFile.Close()

	// create destination Pixi file with compressed layers
	dstPixi := pixi.NewHeader(srcPixi.Header.ByteOrder, srcPixi.Header.OffsetSize)
	summary := &pixi.Pixi{
		Header: dstPixi,
	}

	err = summary.AppendTags(dstFile, srcPixi.AllTags())
	if err != nil {
		fmt.Println("Failed to write tags to destination Pixi file.")
		return
	}

	for _, srcLayer := range srcPixi.Layers {
		opts := []pixi.LayerOption{pixi.WithCompression(compression)}
		if srcLayer.Separated {
			opts = append(opts, pixi.WithPlanar())
		}
		dstLayer := pixi.NewLayer(srcLayer.Name, srcLayer.Dimensions, srcLayer.Channels, opts...)
		srcData := pixi.NewFifoCacheReadLayer(srcStream, srcPixi.Header, srcLayer, 4)

		err = summary.AppendIterativeLayer(dstFile, dstLayer, func(dstIterator pixi.IterativeLayerWriter) error {
			for dstIterator.Next() {
				coord := dstIterator.Coordinate()
				pixel, err := pixi.SampleAt(srcData, coord)
				if err != nil {
					return fmt.Errorf("Failed to read sample from source Pixi file.")
				}
				dstIterator.SetSample(pixel)
			}
			return nil
		})
		if err != nil {
			fmt.Println("Failed to write layer to destination Pixi file.")
			return
		}
	}
}
