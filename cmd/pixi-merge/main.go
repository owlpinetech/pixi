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
	separatedArg := flag.Bool("sep", false, "whether to separate channels of layers in the output file")
	compressionArg := flag.String("comp", "none", "compression type for output file (none, flate, lzw-lsb, lzw-msb)")
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Println("No input files provided")
		return
	}

	if *dstFileName == "" {
		fmt.Println("No destination file name provided")
		return
	}

	compression := pixi.CompressionNone
	switch *compressionArg {
	case "none":
		compression = pixi.CompressionNone
	case "flate":
		compression = pixi.CompressionFlate
	case "lzw-lsb":
		compression = pixi.CompressionLzwLsb
	case "lzw-msb":
		compression = pixi.CompressionLzwMsb
	case "rle8":
		compression = pixi.CompressionRle8
	default:
		fmt.Printf("Unsupported compression type: %s\n", *compressionArg)
		return
	}

	srcFileNames := flag.Args()

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
	targetHeader := &pixi.Header{}
	targetDimensions := []pixi.DimensionSet{}
	srcPixis := []*pixi.Pixi{}
	srcReaders := map[int][]*pixi.TileOrderReadIterator{}
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
			targetLayerCount = len(srcPixi.Layers)
			targetHeader = srcPixi.Header
			for _, layer := range srcPixi.Layers {
				targetDimensions = append(targetDimensions, layer.Dimensions)
			}
		} else if len(srcPixi.Layers) != targetLayerCount {
			fmt.Printf("Source Pixi file '%s' has a different number of layers (%d) than previous files (%d).\n", srcFileNames[srcIndex], len(srcPixi.Layers), targetLayerCount)
			return
		} else {
			for layerInd, layer := range srcPixi.Layers {
				if !slices.Equal(layer.Dimensions, targetDimensions[layerInd]) {
					fmt.Printf("Source Pixi file '%s' has different dimensions for layer %d than previous files.\n", srcFileNames[srcIndex], layerInd)
					return
				}
			}
		}

		if targetHeader.OffsetSize < srcPixi.Header.OffsetSize {
			targetHeader.OffsetSize = srcPixi.Header.OffsetSize
		}

		for layerIndex, layer := range srcPixi.Layers {
			layerReader := pixi.NewTileOrderReadIterator(srcStream, srcPixi.Header, layer)
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
		mergedChannels := pixi.ChannelSet{}
		for _, reader := range layerReaders {
			for _, channel := range reader.Layer().Channels {
				mergedChannels = append(mergedChannels, channel)
			}
		}

		mergedLayer := pixi.NewLayer(
			strings.Join(layerNames[layerIndex], "+"),
			*separatedArg,
			compression,
			targetDimensions[layerIndex],
			mergedChannels,
		)
		previousLayer = mergedLayer

		dstLayerWriter := pixi.NewTileOrderWriteIterator(dstFile, dstPixi, mergedLayer)

		for dstLayerWriter.Next() {
			// advance all readers by one too
			readerAdvanceSuccess := true
			for _, reader := range layerReaders {
				readerAdvanceSuccess = readerAdvanceSuccess && reader.Next()
			}
			if !readerAdvanceSuccess {
				fmt.Println("Failed to advance tile readers for source Pixi files.")
				for _, reader := range layerReaders {
					if reader.Error() != nil {
						fmt.Println("  Reader error:", reader.Error())
					}
				}
				return
			}

			dstIndex := 0
			for _, reader := range layerReaders {
				sample := reader.Sample()
				for _, channel := range sample {
					dstLayerWriter.SetChannel(dstIndex, channel)
					dstIndex += 1
				}
			}
		}

		for _, reader := range layerReaders {
			reader.Done()
		}

		dstLayerWriter.Done()

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

		if dstLayerWriter.Error() != nil {
			fmt.Println("Failed to finalize layer writing to destination Pixi file.")
			return
		}
	}
}
