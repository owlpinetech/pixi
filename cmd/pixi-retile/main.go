package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/owlpinetech/pixi"
	"github.com/owlpinetech/pixi/edit"
	"github.com/owlpinetech/pixi/read"
)

func main() {
	srcFileName := flag.String("src", "", "name of the pixi file to open")
	dstFileName := flag.String("dst", "", "name of the output pixi file")
	layerInd := flag.Int("layer", 0, "index of the layer to retile")
	sizes := flag.String("tiles", "", "comma-separated list of tile sizes for each dimension")
	flag.Parse()

	// convert and verify tile sizes
	sizeStrs := strings.Split(*sizes, ",")
	tileSizes := make([]int, len(sizeStrs))
	for i, sizeStr := range sizeStrs {
		var err error
		tileSizes[i], err = strconv.Atoi(sizeStr)
		if err != nil {
			fmt.Printf("Invalid tile size: %s\n", sizeStr)
			os.Exit(1)
		}
		if tileSizes[i] <= 0 {
			fmt.Printf("Tile size at index %d must be greater than zero\n", i)
			os.Exit(1)
		}
	}

	if len(tileSizes) == 0 {
		fmt.Println("No tile sizes provided")
		os.Exit(1)
	}

	// open source and destination files
	srcFile, err := os.Open(*srcFileName)
	if err != nil {
		fmt.Println("Failed to open source file.")
		os.Exit(1)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(*dstFileName)
	if err != nil {
		fmt.Println("Failed to create destination file.")
		os.Exit(1)
	}
	defer dstFile.Close()

	// read source Pixi file and validate layer index
	srcPixi, err := pixi.ReadPixi(srcFile)
	if err != nil {
		fmt.Println("Failed to read source Pixi file.")
		os.Exit(1)
	}

	if *layerInd < 0 || *layerInd >= len(srcPixi.Layers) {
		fmt.Println("Invalid layer index.")
		os.Exit(1)
	}
	srcLayer := srcPixi.Layers[*layerInd]
	if len(srcLayer.Dimensions) != len(tileSizes) {
		fmt.Println("Number of tile sizes does not match number of dimensions in layer.")
		os.Exit(1)
	}

	// create destination Pixi file with updated layer
	dstPixi := &pixi.PixiHeader{Version: pixi.Version, OffsetSize: srcPixi.Header.OffsetSize, ByteOrder: srcPixi.Header.ByteOrder}

	dstDims := make(pixi.DimensionSet, len(srcLayer.Dimensions))
	for i, dim := range srcLayer.Dimensions {
		dstDims[i] = &pixi.Dimension{
			Name:     dim.Name,
			Size:     dim.Size,
			TileSize: tileSizes[i],
		}
	}
	dstLayer := pixi.NewLayer(srcLayer.Name, srcLayer.Separated, srcLayer.Compression, dstDims, srcLayer.Fields)

	if err := dstPixi.WriteHeader(dstFile); err != nil {
		fmt.Println("Failed to header to write destination Pixi file.")
		os.Exit(1)
	}

	srcData := read.NewLayerReadCache(srcFile, srcPixi.Header, srcLayer, read.NewLfuCacheManager(4))

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
		os.Exit(1)
	}
}
