package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/owlpinetech/pixi"
)

func main() {
	fileName := flag.String("file", "", "name of the pixi file to examine or tag")
	tags := flag.String("tags", "", "tag name to add to the pixi file")
	vals := flag.String("vals", "", "value of the tag to add to the pixi file")
	flag.Parse()

	if *fileName == "" {
		fmt.Println("No pixi file provided")
		os.Exit(1)
	}

	fileInfo, err := os.Stat(*fileName)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fileSize := fileInfo.Size()

	file, err := os.OpenFile(*fileName, os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Failed to open pixi file.")
		os.Exit(1)
	}
	defer file.Close()

	root, err := pixi.ReadPixi(file)
	if err != nil {
		fmt.Println("Failed to read source Pixi file.")
		os.Exit(1)
	}

	allTags := root.AllTags()

	tagNames := strings.Split(*tags, ",")
	tagVals := strings.Split(*vals, ",")

	if len(tagNames) != len(tagVals) {
		fmt.Println("Number of tags and values must match")
		os.Exit(1)
	}

	if *tags == "" || len(tagNames) == 0 {
		fmt.Println("No tag names provided, printing all tags:")
		for k, v := range allTags {
			fmt.Printf("%s => %s\n", k, v)
		}
		return
	}
	for _, tag := range tagNames {
		if _, ok := allTags[tag]; ok {
			fmt.Printf("Tag %s already exists with value %s\n", tag, allTags[tag])
			os.Exit(1)
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

	root.Tags[len(root.Tags)-1].NextTagsStart = fileSize
	if len(root.Tags) > 1 {
		file.Seek(root.Tags[len(root.Tags)-2].NextTagsStart, io.SeekStart)
	} else {
		file.Seek(root.Header.FirstTagsOffset, io.SeekStart)
	}
	err = root.Tags[len(root.Tags)-1].Write(file, root.Header)
	if err != nil {
		fmt.Println("Failed to overwrite previous tag section in Pixi file.")
		os.Exit(1)
	}
	file.Seek(fileSize, io.SeekStart)
	err = newSection.Write(file, root.Header)
	if err != nil {
		fmt.Println("Failed to write new tag section to Pixi file.")
		os.Exit(1)
	}
}
