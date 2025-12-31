package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gracefulearth/gopixi"
)

func main() {
	srcFileName := flag.String("src", "", "path to the pixi file to open")
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
			return
		}
		if tileSizes[i] <= 0 {
			fmt.Printf("Tile size at index %d must be greater than zero\n", i)
			return
		}
	}

	if len(tileSizes) == 0 {
		fmt.Println("No tile sizes provided")
		return
	}

	// open source and destination files
	srcStream, err := gopixi.OpenFileOrHttp(*srcFileName)
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
	srcPixi, err := gopixi.ReadPixi(srcStream)
	if err != nil {
		fmt.Println("Failed to read source Pixi file.")
		return
	}

	if *layerInd < 0 || *layerInd >= len(srcPixi.Layers) {
		fmt.Println("Invalid layer index.")
		return
	}
	srcLayer := srcPixi.Layers[*layerInd]
	if len(srcLayer.Dimensions) != len(tileSizes) {
		fmt.Println("Number of tile sizes does not match number of dimensions in layer.")
		return
	}

	// create destination Pixi file with updated layer
	dstPixi := gopixi.NewHeader(srcPixi.Header.ByteOrder, srcPixi.Header.OffsetSize)
	err = dstPixi.WriteHeader(dstFile)
	if err != nil {
		fmt.Println("Failed to write Pixi header to destination Pixi file.")
		return
	}
	dstSummary := &gopixi.Pixi{
		Header: dstPixi,
	}

	err = dstSummary.AppendTags(dstFile, srcPixi.AllTags())
	if err != nil {
		fmt.Println("Failed to write tags to destination Pixi file.")
		return
	}

	dstDims := make(gopixi.DimensionSet, len(srcLayer.Dimensions))
	for i, dim := range srcLayer.Dimensions {
		dstDims[i] = gopixi.Dimension{
			Name:     dim.Name,
			Size:     dim.Size,
			TileSize: tileSizes[i],
		}
	}
	opts := []gopixi.LayerOption{gopixi.WithCompression(srcLayer.Compression)}
	if srcLayer.Separated {
		opts = append(opts, gopixi.WithPlanar())
	}
	dstLayer := gopixi.NewLayer(srcLayer.Name, dstDims, srcLayer.Channels, opts...)

	srcData := gopixi.NewFifoCacheReadLayer(srcStream, srcPixi.Header, srcLayer, 4)

	err = dstSummary.AppendIterativeLayer(dstFile, dstLayer, func(dstIterator gopixi.IterativeLayerWriter) error {
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
