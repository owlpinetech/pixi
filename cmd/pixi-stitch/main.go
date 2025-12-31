package main

import (
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/gracefulearth/gopixi"
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
		srcStream, err := gopixi.OpenFileOrHttp(srcFileName)
		if err != nil {
			fmt.Printf("Failed to open source Pixi file '%s'.\n", srcFileName)
			return
		}
		defer srcStream.Close()

		srcStreams = append(srcStreams, srcStream)
	}

	targetLayerCount := -1
	targetSeparated := []bool{}
	targetCompressions := []gopixi.Compression{}
	targetHeader := gopixi.Header{}
	targetDimensions := []gopixi.DimensionSet{}
	targetChannels := []gopixi.ChannelSet{}
	srcPixis := []*gopixi.Pixi{}
	srcReaders := map[int][]gopixi.TileAccessLayer{}
	layerNames := map[int][]string{}
	tags := map[string]string{}
	for srcIndex, srcStream := range srcStreams {
		srcPixi, err := gopixi.ReadPixi(srcStream)
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
			layerReader := gopixi.NewFifoCacheReadLayer(srcStream, srcPixi.Header, layer, 16)
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

	dstPixi := gopixi.NewHeader(targetHeader.ByteOrder, targetHeader.OffsetSize)
	err = dstPixi.WriteHeader(dstFile)
	if err != nil {
		fmt.Println("Failed to write Pixi header to destination Pixi file.")
		return
	}
	dstSummary := &gopixi.Pixi{
		Header: dstPixi,
	}

	err = dstSummary.AppendTags(dstFile, tags)
	if err != nil {
		fmt.Println("Failed to write tags to destination Pixi file.")
		return
	}

	for layerIndex, layerReaders := range srcReaders {
		opts := []gopixi.LayerOption{gopixi.WithCompression(targetCompressions[layerIndex])}
		if targetSeparated[layerIndex] {
			opts = append(opts, gopixi.WithPlanar())
		}
		mergedLayer := gopixi.NewLayer(
			strings.Join(layerNames[layerIndex], "+"),
			targetDimensions[layerIndex],
			targetChannels[layerIndex],
			opts...,
		)

		err = dstSummary.AppendIterativeLayer(dstFile, mergedLayer, func(dstLayerWriter gopixi.IterativeLayerWriter) error {
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
				sample, err := gopixi.SampleAt(layerReaders[srcReaderIndex], coord)
				if err != nil {
					return fmt.Errorf("Failed to retrieve sample from source Pixi files: %v", err)
				}

				dstLayerWriter.SetSample(sample)
			}
			return nil
		})
		if err != nil {
			fmt.Println("Failed to write layer to destination Pixi file.")
			return
		}
	}
}
