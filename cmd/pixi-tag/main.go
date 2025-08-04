package main

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/owlpinetech/pixi"
	"github.com/owlpinetech/pixi/read"
)

func main() {
	pixiPath := flag.String("path", "", "path of the pixi file to examine or tag")
	tags := flag.String("tags", "", "tag name to add to the pixi file")
	vals := flag.String("vals", "", "value of the tag to add to the pixi file")
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
		pixiStream, err = read.OpenBufferedHttp(pixiUrl, nil)
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

	root, err := pixi.ReadPixi(pixiStream)
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

	if *tags == "" || len(tagNames) == 0 {
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
	newSection := &pixi.TagSection{
		Tags:          newTags,
		NextTagsStart: 0,
	}

	fileInfo, err := editFile.Stat()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fileSize := fileInfo.Size()

	root.Tags[len(root.Tags)-1].NextTagsStart = fileSize
	if len(root.Tags) > 1 {
		editFile.Seek(root.Tags[len(root.Tags)-2].NextTagsStart, io.SeekStart)
	} else {
		editFile.Seek(root.Header.FirstTagsOffset, io.SeekStart)
	}
	err = root.Tags[len(root.Tags)-1].Write(editFile, root.Header)
	if err != nil {
		fmt.Println("Failed to overwrite previous tag section in Pixi file.")
		return
	}
	editFile.Seek(fileSize, io.SeekStart)
	err = newSection.Write(editFile, root.Header)
	if err != nil {
		fmt.Println("Failed to write new tag section to Pixi file.")
		return
	}
}
