package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gracefulearth/gopixi"
)

func main() {
	srcFileName := flag.String("src", "", "path to the pixi file to open")
	dstFileName := flag.String("dst", "", "name of the output pixi file")
	method := flag.String("method", "flate", "compression method to use (flate, lzw_lsb, lzw_msb, none)")
	flag.Parse()

	// determine compression method
	var compression gopixi.Compression
	switch *method {
	case "flate":
		compression = gopixi.CompressionFlate
	case "lzw_lsb":
		compression = gopixi.CompressionLzwLsb
	case "lzw_msb":
		compression = gopixi.CompressionLzwMsb
	case "rle8":
		compression = gopixi.CompressionRle8
	case "none":
		compression = gopixi.CompressionNone
	default:
		fmt.Println("Invalid compression method. Must be one of: flate, lzw_lsb, lzw_msb, none")
		return
	}

	// open source and destination files
	srcStream, err := gopixi.OpenFileOrHttp(*srcFileName)
	if err != nil {
		fmt.Println("Failed to open source Pixi file:", err)
		return
	}
	defer srcStream.Close()

	// read source Pixi file and validate layer index
	srcPixi, err := gopixi.ReadPixi(srcStream)
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
	dstPixi := gopixi.NewHeader(srcPixi.Header.ByteOrder, srcPixi.Header.OffsetSize)
	summary := &gopixi.Pixi{
		Header: dstPixi,
	}

	err = summary.AppendTags(dstFile, srcPixi.AllTags())
	if err != nil {
		fmt.Println("Failed to write tags to destination Pixi file.")
		return
	}

	for _, srcLayer := range srcPixi.Layers {
		opts := []gopixi.LayerOption{gopixi.WithCompression(compression)}
		if srcLayer.Separated {
			opts = append(opts, gopixi.WithPlanar())
		}
		dstLayer := gopixi.NewLayer(srcLayer.Name, srcLayer.Dimensions, srcLayer.Channels, opts...)
		srcData := gopixi.NewFifoCacheReadLayer(srcStream, srcPixi.Header, srcLayer, 4)

		iterator := gopixi.NewTileOrderWriteIterator(dstFile, dstPixi, dstLayer)
		err = summary.AppendIterativeLayer(dstFile, dstLayer, iterator, func(dstIterator gopixi.IterativeLayerWriter) error {
			for dstIterator.Next() {
				coord := dstIterator.Coordinate()
				pixel, err := gopixi.SampleAt(srcData, coord)
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
