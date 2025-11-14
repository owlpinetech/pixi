package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/owlpinetech/pixi"
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
		fmt.Println("Failed to write destination Pixi file.")
		return
	}
}
