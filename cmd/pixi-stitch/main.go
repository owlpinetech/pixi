package main

import (
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/owlpinetech/pixi"
)

func main() {
	dstFileName := flag.String("dst", "", "name of the output pixi file")
	stitchDimension := flag.Int("dim", 0, "dimension index to stitch along")
	flag.Parse()

	if len(flag.Args()) < 2 {
		fmt.Println("Not enough input files provided, require at least 2 to stitch")
		return
	}
	if len(flag.Args()) > 2 {
		fmt.Println("More than 2 input files provided, stitching more than 2 files is not yet supported, additional files will be ignored")
	}

	if *dstFileName == "" {
		fmt.Println("No destination file name provided")
		return
	}

	srcFileNames := flag.Args()[:2]

	// open input files
	srcStreams := []io.ReadSeekCloser{}
	for _, srcFileName := range srcFileNames {
		srcStream, err := pixi.OpenFileOrHttp(srcFileName)
		if err != nil {
			fmt.Printf("Failed to open source Pixi file '%s'.\n", srcFileName)
			return
		}
		defer srcStream.Close()

		srcStreams = append(srcStreams, srcStream)
	}

	targetLayerCount := -1
	targetSeparated := []bool{}
	targetCompressions := []pixi.Compression{}
	targetHeader := &pixi.Header{}
	targetDimensions := []pixi.DimensionSet{}
	targetChannels := []pixi.ChannelSet{}
	srcPixis := []*pixi.Pixi{}
	srcReaders := map[int][]pixi.TileAccessLayer{}
	layerNames := map[int][]string{}
	tags := map[string]string{}
	for srcIndex, srcStream := range srcStreams {
		srcPixi, err := pixi.ReadPixi(srcStream)
		if err != nil {
			fmt.Printf("Failed to read source Pixi file '%s'.\n", srcFileNames[srcIndex])
			return
		}

		maps.Copy(tags, srcPixi.AllTags())

		srcPixis = append(srcPixis, srcPixi)

		if targetLayerCount == -1 {
			if *stitchDimension < 0 || *stitchDimension >= len(srcPixi.Layers[0].Dimensions) {
				fmt.Printf("Stitch dimension %d is out of bounds for source Pixi file '%s'.\n", *stitchDimension, srcFileNames[srcIndex])
				return
			}
			targetLayerCount = len(srcPixi.Layers)
			targetHeader = srcPixi.Header
			for _, layer := range srcPixi.Layers {
				targetDimensions = append(targetDimensions, slices.Clone(layer.Dimensions))
				targetChannels = append(targetChannels, slices.Clone(layer.Channels))
				targetCompressions = append(targetCompressions, layer.Compression)
				targetSeparated = append(targetSeparated, layer.Separated)
			}
		} else if len(srcPixi.Layers) != targetLayerCount {
			fmt.Printf("Source Pixi file '%s' has a different number of layers (%d) than previous files (%d).\n", srcFileNames[srcIndex], len(srcPixi.Layers), targetLayerCount)
			return
		} else {
			for layerInd, layer := range srcPixi.Layers {
				if len(layer.Dimensions) != len(targetDimensions[layerInd]) {
					fmt.Printf("Source Pixi file '%s' has different number of dimensions for layer %d than previous files.\n", srcFileNames[srcIndex], layerInd)
					return
				}
				for channelInd, channel := range layer.Channels {
					if channel.Type != targetChannels[layerInd][channelInd].Type {
						fmt.Printf("Source Pixi file '%s' has different channel types/sizes for layer %d than previous files.\n", srcFileNames[srcIndex], layerInd)
						return
					}
				}
				targetDimensions[layerInd][*stitchDimension].Size += layer.Dimensions[*stitchDimension].Size
			}
		}

		// TODO: from size estimate of all source files, decide on an appropriate offset size
		if targetHeader.OffsetSize < srcPixi.Header.OffsetSize {
			targetHeader.OffsetSize = srcPixi.Header.OffsetSize
		}

		for layerIndex, layer := range srcPixi.Layers {
			layerReader := pixi.NewFifoCacheReadLayer(srcStream, srcPixi.Header, layer, 16)
			srcReaders[layerIndex] = append(srcReaders[layerIndex], layerReader)
			if !slices.Contains(layerNames[layerIndex], layer.Name) {
				layerNames[layerIndex] = append(layerNames[layerIndex], layer.Name)
			}
		}
	}

	// create destination file & scaffold pixi
	dstFile, err := os.Create(*dstFileName)
	if err != nil {
		fmt.Println("Failed to create destination file.")
		return
	}
	defer dstFile.Close()

	dstPixi := &pixi.Header{
		Version:    pixi.Version,
		OffsetSize: targetHeader.OffsetSize,
		ByteOrder:  targetHeader.ByteOrder,
	}
	err = dstPixi.WriteHeader(dstFile)
	if err != nil {
		fmt.Println("Failed to write Pixi header to destination Pixi file.")
		return
	}

	tagSection := pixi.TagSection{Tags: tags}
	err = tagSection.Write(dstFile, dstPixi)
	if err != nil {
		fmt.Println("Failed to write tags to destination Pixi file.")
		return
	}

	previousOffset := dstPixi.FirstLayerOffset
	var previousLayer *pixi.Layer
	for layerIndex, layerReaders := range srcReaders {
		mergedLayer := pixi.NewLayer(
			strings.Join(layerNames[layerIndex], "+"),
			targetSeparated[layerIndex],
			targetCompressions[layerIndex],
			targetDimensions[layerIndex],
			targetChannels[layerIndex],
		)
		previousLayer = mergedLayer

		dstLayerWriter := pixi.NewTileOrderWriteIterator(dstFile, dstPixi, mergedLayer)

		for dstLayerWriter.Next() {
			coord := dstLayerWriter.Coordinate()

			// determine which source reader to pull from
			stitchPos := coord[*stitchDimension]
			srcReaderIndex := 0
			for ; srcReaderIndex < len(layerReaders)-1; srcReaderIndex++ {
				if stitchPos < layerReaders[srcReaderIndex].Layer().Dimensions[*stitchDimension].Size {
					break
				}
				stitchPos -= layerReaders[srcReaderIndex].Layer().Dimensions[*stitchDimension].Size
			}
			// adjust coordinate to source reader space
			coord[*stitchDimension] = stitchPos
			sample, err := pixi.SampleAt(layerReaders[srcReaderIndex], coord)
			if err != nil {
				fmt.Println("Error at coordinate:", coord, "original:", dstLayerWriter.Coordinate(), "dimensions:", layerReaders[srcReaderIndex].Layer().Dimensions)
				fmt.Println("Failed to retrieve sample from source Pixi files: ", err)
				return
			}

			dstLayerWriter.SetSample(sample)
		}

		dstLayerWriter.Done()
		if dstLayerWriter.Error() != nil {
			fmt.Println("Failed to finalize layer writing to destination Pixi file.")
			return
		}

		offset, err := dstFile.Seek(0, io.SeekCurrent)
		if err != nil {
			fmt.Println("Failed to seek in destination Pixi file.")
			return
		}
		if previousLayer != nil {
			previousLayer.NextLayerStart = offset
			previousLayer.OverwriteHeader(dstFile, dstPixi, previousOffset)
		} else {
			dstPixi.OverwriteOffsets(dstFile, offset, int64(dstPixi.DiskSize()))
		}
		previousOffset = offset

		err = mergedLayer.WriteHeader(dstFile, dstPixi)
		if err != nil {
			fmt.Println("Failed to write layer header to destination Pixi file.")
			return
		}
	}
}
