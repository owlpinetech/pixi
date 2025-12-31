package main

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/gracefulearth/gopixi"
)

func main() {
	pixiPath := flag.String("path", "", "path of the pixi file to examine or tag")
	tags := flag.String("tags", "", "comma-separated tag names to add to the pixi file")
	vals := flag.String("vals", "", "comma-seperated values of the tags to add to the pixi file")
	flag.Parse()

	if *pixiPath == "" {
		fmt.Println("No pixi file provided")
		os.Exit(1)
	}

	var editFile *os.File
	var pixiStream io.ReadSeeker
	if strings.HasPrefix(*pixiPath, "http://") || strings.HasPrefix(*pixiPath, "https://") {
		pixiUrl, err := url.Parse(*pixiPath)
		if err != nil {
			fmt.Println("Invalid URL:", err)
			return
		}
		pixiStream, err = gopixi.OpenBufferedHttp(pixiUrl, nil)
		if err != nil {
			fmt.Println("Failed to open remote Pixi file:", err)
			return
		}

		if *tags != "" || *vals != "" {
			fmt.Println("Editing remote Pixi files over HTTP is not supported.")
			return
		}
	} else {
		file, err := os.OpenFile(*pixiPath, os.O_RDWR, 0644)
		if err != nil {
			fmt.Println("Failed to open pixi file.")
			return
		}
		defer file.Close()

		editFile = file
		pixiStream = file
	}

	root, err := gopixi.ReadPixi(pixiStream)
	if err != nil {
		fmt.Println("Failed to read source Pixi file.", err)
		return
	}

	allTags := root.AllTags()

	tagNames := strings.Split(*tags, ",")
	tagVals := strings.Split(*vals, ",")

	if len(tagNames) != len(tagVals) {
		fmt.Println("Number of tags and values must match")
		return
	}

	if len(tagNames) == 0 {
		fmt.Println("Listing Pixi tags:")
		for k, v := range allTags {
			fmt.Printf("%s => %s\n", k, v)
		}
		return
	}
	for _, tag := range tagNames {
		if _, ok := allTags[tag]; ok {
			fmt.Printf("Tag %s already exists with value %s\n", tag, allTags[tag])
			return
		}
	}

	newTags := make(map[string]string)
	for i, tag := range tagNames {
		newTags[tag] = tagVals[i]
	}

	err = root.AppendTags(editFile, newTags)
	if err != nil {
		fmt.Println("Failed to append tags to Pixi file:", err)
		return
	}
}
