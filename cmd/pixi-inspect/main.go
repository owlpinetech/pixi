package main

import (
	"flag"
	"fmt"

	"github.com/owlpinetech/pixi"
)

func main() {
	pixiPath := flag.String("path", "", "path to the pixi file to open, e.g. /path/to/file.pixi or http://example.com/file.pixi")
	flag.Parse()

	if *pixiPath == "" {
		fmt.Println("must specify a Pixi file to inspect")
		return
	}

	pixiStream, err := pixi.OpenFileOrHttp(*pixiPath)
	if err != nil {
		fmt.Println("Failed to open source Pixi file:", err)
		return
	}
	defer pixiStream.Close()

	summary, err := pixi.ReadPixi(pixiStream)

	fmt.Printf("Inspecting %s\n", *pixiPath)
	fmt.Printf("\tVersion: %d\n", summary.Header.Version)
	fmt.Printf("\tOffset size: %d\n", summary.Header.OffsetSize)
	fmt.Printf("\tByte order: %s\n", summary.Header.ByteOrder)
	fmt.Printf("Tag Sections: %d\n", len(summary.Tags))
	for sectionInd, section := range summary.Tags {
		fmt.Printf("\tSection %d\n", sectionInd)
		for k, v := range section.Tags {
			fmt.Printf("\t\t%s: %s\n", k, v)
		}
	}
	fmt.Printf("Layers: %d\n", len(summary.Layers))
	for layerInd, layer := range summary.Layers {
		fmt.Printf("\tLayer %d: %s\n", layerInd, layer.Name)
		fmt.Printf("\t\tSeparated: %v\n", layer.Separated)
		fmt.Printf("\t\tCompression: %s\n", layer.Compression)
		fmt.Printf("\t\tDimensions: %d\n", len(layer.Dimensions))
		for dimInd, dim := range layer.Dimensions {
			fmt.Printf("\t\t\tDim %d (%s): %d / %d (%d tiles)\n", dimInd, dim.Name, dim.Size, dim.TileSize, dim.Tiles())
		}
		fmt.Printf("\t\tFields: %d\n", len(layer.Fields))
		for fieldInd, field := range layer.Fields {
			fmt.Printf("\t\t\tField %d (%s) : %s\n", fieldInd, field.Name, field.Type)
		}
	}

	if err != nil {
		fmt.Println(err)
		return
	}
}
