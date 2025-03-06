package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/owlpinetech/pixi"
)

func main() {
	fileName := flag.String("file", "", "name of the pixi file to open")
	flag.Parse()

	if *fileName == "" {
		fmt.Println("must specify a Pixi file to inspect")
		os.Exit(-1)
	}

	pixiFile, err := os.Open(*fileName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer pixiFile.Close()

	summary, err := pixi.ReadPixi(pixiFile)

	fmt.Printf("Inspecting %s\n", *fileName)
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
		os.Exit(1)
	}
}
