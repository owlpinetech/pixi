package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
	"strconv"

	"github.com/owlpinetech/pixi"
)

// DecimationMethod defines the method for combining higher resolution pixels
type DecimationMethod int

const (
	MethodMax DecimationMethod = iota
	MethodMin
	MethodMean
	MethodMedian
	MethodFirst  // Take the first pixel (top-left for 2D)
	MethodCenter // Take the center pixel from the source region
)

func (m DecimationMethod) String() string {
	switch m {
	case MethodMax:
		return "max"
	case MethodMin:
		return "min"
	case MethodMean:
		return "mean"
	case MethodMedian:
		return "median"
	case MethodFirst:
		return "first"
	case MethodCenter:
		return "center"
	default:
		return "unknown"
	}
}

func parseMethod(s string) (DecimationMethod, error) {
	switch s {
	case "max":
		return MethodMax, nil
	case "min":
		return MethodMin, nil
	case "mean":
		return MethodMean, nil
	case "median":
		return MethodMedian, nil
	case "first":
		return MethodFirst, nil
	case "center":
		return MethodCenter, nil
	default:
		return MethodMax, fmt.Errorf("invalid decimation method: %s", s)
	}
}

func main() {
	srcFileName := flag.String("src", "", "path to the pixi file to open")
	dstFileName := flag.String("dst", "", "name of the output pixi file")
	methodArg := flag.String("method", "mean", "decimation method (max, min, mean, median, first, center)")
	factorArg := flag.String("factor", "0.5", "decimation factor as percentage (0.0-1.0)")
	flag.Parse()

	if *srcFileName == "" || *dstFileName == "" {
		fmt.Println("Both src and dst must be specified")
		flag.Usage()
		return
	}

	// Parse decimation method
	decimationMethod, err := parseMethod(*methodArg)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Parse factor
	factor, err := strconv.ParseFloat(*factorArg, 64)
	if err != nil || factor <= 0 || factor > 1 {
		fmt.Println("Decimation factor must be a number between 0 and 1")
		return
	}

	// Open source file
	srcStream, err := pixi.OpenFileOrHttp(*srcFileName)
	if err != nil {
		fmt.Println("Failed to open source Pixi file:", err)
		return
	}
	defer srcStream.Close()

	// Create destination file
	dstFile, err := os.Create(*dstFileName)
	if err != nil {
		fmt.Println("Failed to create destination file:", err)
		return
	}
	defer dstFile.Close()

	// Read source Pixi file
	srcPixi, err := pixi.ReadPixi(srcStream)
	if err != nil {
		fmt.Println("Failed to read source Pixi file:", err)
		return
	}

	// Create destination Pixi file
	dstPixi := &pixi.PixiHeader{
		Version:    pixi.Version,
		OffsetSize: srcPixi.Header.OffsetSize,
		ByteOrder:  srcPixi.Header.ByteOrder,
	}

	// Process each layer
	for layerIdx, srcLayer := range srcPixi.Layers {

		// Calculate new dimensions
		newDims := make(pixi.DimensionSet, len(srcLayer.Dimensions))
		for i, dim := range srcLayer.Dimensions {
			newSize := max(int(math.Ceil(float64(dim.Size)*factor)), 1)
			// Keep tile size proportional but ensure it's at least 1 and not larger than the new size
			newTileSize := min(max(int(math.Ceil(float64(dim.TileSize)*factor)), 1), newSize)
			newDims[i] = pixi.Dimension{
				Name:     dim.Name,
				Size:     newSize,
				TileSize: newTileSize,
			}
		}

		// Create destination layer
		dstLayer := pixi.NewLayer(
			srcLayer.Name+"_decimated",
			srcLayer.Separated,
			srcLayer.Compression,
			newDims,
			srcLayer.Fields,
		)

		// Create cached reader for source layer
		srcData := pixi.NewFifoCacheReadLayer(srcStream, srcPixi.Header, srcLayer, 4)

		// Write header
		err = dstPixi.WriteHeader(dstFile)
		if err != nil {
			fmt.Printf("Failed to write Pixi header: %v\n", err)
			return
		}

		// Write tags section
		tagsOffset, err := dstFile.Seek(0, io.SeekCurrent)
		if err != nil {
			fmt.Printf("Failed to seek in destination file: %v\n", err)
			return
		}
		tagSection := pixi.TagSection{Tags: srcPixi.AllTags(), NextTagsStart: 0}
		err = tagSection.Write(dstFile, dstPixi)
		if err != nil {
			fmt.Printf("Failed to write tags: %v\n", err)
			return
		}

		// Create write iterator for destination layer
		dstIterator := pixi.NewTileOrderWriteIterator(dstFile, dstPixi, dstLayer)

		// Decimate the data
		err = decimateLayer(srcData, dstIterator, srcLayer.Dimensions, decimationMethod, factor)
		if err != nil {
			fmt.Printf("Failed to decimate layer %d: %v\n", layerIdx, err)
			return
		}

		dstIterator.Done()
		if dstIterator.Error() != nil {
			fmt.Printf("Failed during tile writing iteration for layer %d: %v\n", layerIdx, dstIterator.Error())
			return
		}

		// Get current position for layer offset
		firstLayerOffset, err := dstFile.Seek(0, io.SeekCurrent)
		if err != nil {
			fmt.Printf("Failed to seek in destination file: %v\n", err)
			return
		}

		// Update offsets
		err = dstPixi.OverwriteOffsets(dstFile, firstLayerOffset, tagsOffset)
		if err != nil {
			fmt.Printf("Failed to overwrite offsets: %v\n", err)
			return
		}

		// Write layer header
		err = dstLayer.WriteHeader(dstFile, dstPixi)
		if err != nil {
			fmt.Printf("Failed to write layer %d header: %v\n", layerIdx, err)
			return
		}
	}
}

func decimateLayer(srcData pixi.TileAccessLayer, dstIterator pixi.IterativeLayerWriter, srcDims pixi.DimensionSet, method DecimationMethod, factor float64) error {
	// Iterate through each output sample
	for dstIterator.Next() {
		dstCoord := dstIterator.Coordinate()

		// Calculate corresponding region in source
		srcSamples, err := collectSourceSamples(srcData, dstCoord, srcDims, factor)
		if err != nil {
			return fmt.Errorf("failed to collect source samples: %v", err)
		}

		// Apply decimation method - we should always have at least one sample
		if len(srcSamples) == 0 {
			return fmt.Errorf("no source samples found for destination coordinate %v", dstCoord)
		}
		decimatedSample := applySampleDecimation(srcSamples, dstIterator.Layer().Fields, method)
		dstIterator.SetSample(decimatedSample)
	}

	return nil
}

func collectSourceSamples(srcData pixi.TileAccessLayer, dstCoord pixi.SampleCoordinate, srcDims pixi.DimensionSet, factor float64) ([]pixi.Sample, error) {
	var samples []pixi.Sample

	// Calculate the number of source samples per destination sample in each dimension
	samplesPerDst := make([]int, len(dstCoord))
	for i := range dstCoord {
		samplesPerDst[i] = max(1, int(math.Ceil(1.0/factor)))
	}

	// Calculate the source region center for this destination coordinate
	srcCenter := make(pixi.SampleCoordinate, len(dstCoord))
	for i, dstPos := range dstCoord {
		// Map destination coordinate to source center
		srcCenterFloat := (float64(dstPos) + 0.5) / factor
		srcCenter[i] = int(math.Round(srcCenterFloat))
	}

	// Calculate the source region bounds around the center
	srcStart := make(pixi.SampleCoordinate, len(dstCoord))
	srcEnd := make(pixi.SampleCoordinate, len(dstCoord))

	for i := range dstCoord {
		halfSize := samplesPerDst[i] / 2
		// Clamp to source bounds
		srcStart[i] = max(0, srcCenter[i]-halfSize)
		srcEnd[i] = min(srcDims[i].Size, srcStart[i]+samplesPerDst[i])

		// Ensure we have at least one sample
		if srcStart[i] >= srcEnd[i] {
			if srcEnd[i] < srcDims[i].Size {
				srcEnd[i] = srcStart[i] + 1
			} else if srcStart[i] > 0 {
				srcStart[i] = srcEnd[i] - 1
			}
		}
	}

	// Collect all samples in the source region
	err := collectSamplesInRegion(srcData, srcStart, srcEnd, srcDims, &samples)
	if err != nil {
		return nil, err
	}

	return samples, nil
}

func collectSamplesInRegion(srcData pixi.TileAccessLayer, start, end pixi.SampleCoordinate, dims pixi.DimensionSet, samples *[]pixi.Sample) error {
	// Recursive function to iterate through N-dimensional region
	coord := make(pixi.SampleCoordinate, len(start))
	copy(coord, start)

	return collectSamplesRecursive(srcData, coord, start, end, dims, 0, samples)
}

func collectSamplesRecursive(srcData pixi.TileAccessLayer, coord, start, end pixi.SampleCoordinate, dims pixi.DimensionSet, dimIndex int, samples *[]pixi.Sample) error {
	if dimIndex >= len(coord) {
		// We've filled all dimensions, collect this sample
		if dims.ContainsCoordinate(coord) {
			sample, err := pixi.SampleAt(srcData, coord)
			if err != nil {
				return err
			}
			*samples = append(*samples, sample)
		}
		return nil
	}

	// Iterate through this dimension
	for coord[dimIndex] = start[dimIndex]; coord[dimIndex] < end[dimIndex]; coord[dimIndex]++ {
		err := collectSamplesRecursive(srcData, coord, start, end, dims, dimIndex+1, samples)
		if err != nil {
			return err
		}
	}

	return nil
}

func applySampleDecimation(samples []pixi.Sample, fields pixi.FieldSet, method DecimationMethod) pixi.Sample {
	if len(samples) == 1 {
		return samples[0]
	}

	// Apply method per field
	result := make(pixi.Sample, len(fields))
	for fieldIdx, field := range fields {
		result[fieldIdx] = applyFieldDecimation(samples, fieldIdx, field, method)
	}

	return result
}

func applyFieldDecimation(samples []pixi.Sample, fieldIdx int, field pixi.Field, method DecimationMethod) any {
	switch method {
	case MethodFirst:
		return samples[0][fieldIdx]
	case MethodCenter:
		return samples[len(samples)/2][fieldIdx]
	case MethodMax:
		return findMaxField(samples, fieldIdx, field)
	case MethodMin:
		return findMinField(samples, fieldIdx, field)
	case MethodMean:
		return calculateMeanField(samples, fieldIdx)
	case MethodMedian:
		return calculateMedianField(samples, fieldIdx, field)
	default:
		return samples[0][fieldIdx]
	}
}

func findMaxField(samples []pixi.Sample, fieldIdx int, field pixi.Field) any {
	maxVal := samples[0][fieldIdx]

	for i := 1; i < len(samples); i++ {
		val := samples[i][fieldIdx]
		if field.Type.CompareValues(val, maxVal) > 0 {
			maxVal = val
		}
	}

	return maxVal
}

func findMinField(samples []pixi.Sample, fieldIdx int, field pixi.Field) any {
	minVal := samples[0][fieldIdx]

	for i := 1; i < len(samples); i++ {
		val := samples[i][fieldIdx]
		if field.Type.CompareValues(val, minVal) < 0 {
			minVal = val
		}
	}

	return minVal
}

func calculateMeanField(samples []pixi.Sample, fieldIdx int) any {
	first := samples[0][fieldIdx]

	switch first.(type) {
	case float32:
		var sum float64
		for _, sample := range samples {
			sum += float64(sample[fieldIdx].(float32))
		}
		return float32(sum / float64(len(samples)))
	case float64:
		var sum float64
		for _, sample := range samples {
			sum += sample[fieldIdx].(float64)
		}
		return sum / float64(len(samples))
	case int8:
		var sum int64
		for _, sample := range samples {
			sum += int64(sample[fieldIdx].(int8))
		}
		return int8(sum / int64(len(samples)))
	case int16:
		var sum int64
		for _, sample := range samples {
			sum += int64(sample[fieldIdx].(int16))
		}
		return int16(sum / int64(len(samples)))
	case int32:
		var sum int64
		for _, sample := range samples {
			sum += int64(sample[fieldIdx].(int32))
		}
		return int32(sum / int64(len(samples)))
	case int64:
		var sum int64
		for _, sample := range samples {
			sum += sample[fieldIdx].(int64)
		}
		return sum / int64(len(samples))
	case uint8:
		var sum uint64
		for _, sample := range samples {
			sum += uint64(sample[fieldIdx].(uint8))
		}
		return uint8(sum / uint64(len(samples)))
	case uint16:
		var sum uint64
		for _, sample := range samples {
			sum += uint64(sample[fieldIdx].(uint16))
		}
		return uint16(sum / uint64(len(samples)))
	case uint32:
		var sum uint64
		for _, sample := range samples {
			sum += uint64(sample[fieldIdx].(uint32))
		}
		return uint32(sum / uint64(len(samples)))
	case uint64:
		var sum uint64
		for _, sample := range samples {
			sum += sample[fieldIdx].(uint64)
		}
		return sum / uint64(len(samples))
	default:
		// For unsupported types, return the first value
		return first
	}
}

func calculateMedianField(samples []pixi.Sample, fieldIdx int, field pixi.Field) any {
	values := make([]any, len(samples))
	for i, sample := range samples {
		values[i] = sample[fieldIdx]
	}

	slices.SortFunc(values, field.Type.CompareValues)

	mid := len(values) / 2
	if len(values)%2 == 0 {
		return averageTwo(values[mid], values[mid+1])
	} else {
		return values[mid]
	}
}

func averageTwo(a, b any) any {
	switch av := a.(type) {
	case float32:
		bv := b.(float32)
		return (av + bv) / 2
	case float64:
		bv := b.(float64)
		return (av + bv) / 2
	case int8:
		bv := b.(int8)
		return (av + bv) / 2
	case int16:
		bv := b.(int16)
		return (av + bv) / 2
	case int32:
		bv := b.(int32)
		return (av + bv) / 2
	case int64:
		bv := b.(int64)
		return (av + bv) / 2
	case uint8:
		bv := b.(uint8)
		return (av + bv) / 2
	case uint16:
		bv := b.(uint16)
		return (av + bv) / 2
	case uint32:
		bv := b.(uint32)
		return (av + bv) / 2
	case uint64:
		bv := b.(uint64)
		return (av + bv) / 2
	default:
		return a // Return first value for unsupported types
	}
}
