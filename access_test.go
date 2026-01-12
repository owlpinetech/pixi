package gopixi

import (
	"encoding/binary"
	"testing"

	"github.com/gracefulearth/gopixi/internal/buffer"
)

// setupBenchmarkLayer creates a MemoryLayer with the specified configuration for benchmarking
func setupBenchmarkLayer(b *testing.B, dims DimensionSet, channels ChannelSet, separated bool) *MemoryLayer {
	b.Helper()

	header := Header{
		Version:    Version,
		OffsetSize: 4,
		ByteOrder:  binary.LittleEndian,
	}

	var layerOpts []LayerOption
	if separated {
		layerOpts = append(layerOpts, WithPlanar())
	}

	layer := NewLayer("benchmark", dims, channels, layerOpts...)
	buf := buffer.NewBuffer(1000000) // Large buffer to avoid reallocations

	memLayer := NewMemoryLayer(buf, header, layer)

	// Pre-populate some test data to ensure realistic performance
	coord := SampleCoordinate{dims[0].Size / 2, dims[1].Size / 2}
	sample := make(Sample, len(channels))
	for i, ch := range channels {
		switch ch.Type {
		case ChannelUint8:
			sample[i] = uint8(42 + i)
		case ChannelUint16:
			sample[i] = uint16(4200 + i)
		case ChannelUint32:
			sample[i] = uint32(420000 + i)
		case ChannelFloat32:
			sample[i] = float32(42.5 + float32(i))
		case ChannelFloat64:
			sample[i] = float64(42.5 + float64(i))
		}
	}

	err := SetSampleAt(memLayer, coord, sample)
	if err != nil {
		b.Fatal(err)
	}

	return memLayer
}

// Benchmark SampleAt vs ChannelAt with single channel, interleaved layout
func BenchmarkSampleAt_SingleChannel_Interleaved(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 100, TileSize: 10},
		{Name: "y", Size: 100, TileSize: 10},
	}
	channels := ChannelSet{
		{Name: "value", Type: ChannelFloat32},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, false)
	coord := SampleCoordinate{50, 50}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := SampleAt(memLayer, coord)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkChannelAt_SingleChannel_Interleaved(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 100, TileSize: 10},
		{Name: "y", Size: 100, TileSize: 10},
	}
	channels := ChannelSet{
		{Name: "value", Type: ChannelFloat32},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, false)
	coord := SampleCoordinate{50, 50}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ChannelAt(memLayer, coord, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark SampleAt vs ChannelAt with single channel, separated layout
func BenchmarkSampleAt_SingleChannel_Separated(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 100, TileSize: 10},
		{Name: "y", Size: 100, TileSize: 10},
	}
	channels := ChannelSet{
		{Name: "value", Type: ChannelFloat32},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, true)
	coord := SampleCoordinate{50, 50}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := SampleAt(memLayer, coord)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkChannelAt_SingleChannel_Separated(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 100, TileSize: 10},
		{Name: "y", Size: 100, TileSize: 10},
	}
	channels := ChannelSet{
		{Name: "value", Type: ChannelFloat32},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, true)
	coord := SampleCoordinate{50, 50}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ChannelAt(memLayer, coord, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark SampleAt vs ChannelAt with multiple channels (4), interleaved layout
func BenchmarkSampleAt_MultiChannel_Interleaved(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 100, TileSize: 10},
		{Name: "y", Size: 100, TileSize: 10},
	}
	channels := ChannelSet{
		{Name: "red", Type: ChannelUint8},
		{Name: "green", Type: ChannelUint8},
		{Name: "blue", Type: ChannelUint8},
		{Name: "alpha", Type: ChannelUint8},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, false)
	coord := SampleCoordinate{50, 50}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := SampleAt(memLayer, coord)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkChannelAt_MultiChannel_Interleaved(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 100, TileSize: 10},
		{Name: "y", Size: 100, TileSize: 10},
	}
	channels := ChannelSet{
		{Name: "red", Type: ChannelUint8},
		{Name: "green", Type: ChannelUint8},
		{Name: "blue", Type: ChannelUint8},
		{Name: "alpha", Type: ChannelUint8},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, false)
	coord := SampleCoordinate{50, 50}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Access the second channel (green) to test channel offset calculations
		_, err := ChannelAt(memLayer, coord, 1)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark SampleAt vs ChannelAt with multiple channels (4), separated layout
func BenchmarkSampleAt_MultiChannel_Separated(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 100, TileSize: 10},
		{Name: "y", Size: 100, TileSize: 10},
	}
	channels := ChannelSet{
		{Name: "red", Type: ChannelUint8},
		{Name: "green", Type: ChannelUint8},
		{Name: "blue", Type: ChannelUint8},
		{Name: "alpha", Type: ChannelUint8},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, true)
	coord := SampleCoordinate{50, 50}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := SampleAt(memLayer, coord)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkChannelAt_MultiChannel_Separated(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 100, TileSize: 10},
		{Name: "y", Size: 100, TileSize: 10},
	}
	channels := ChannelSet{
		{Name: "red", Type: ChannelUint8},
		{Name: "green", Type: ChannelUint8},
		{Name: "blue", Type: ChannelUint8},
		{Name: "alpha", Type: ChannelUint8},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, true)
	coord := SampleCoordinate{50, 50}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Access the second channel (green)
		_, err := ChannelAt(memLayer, coord, 1)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark with larger dimensions (2D)
func BenchmarkSampleAt_LargeDimensions_2D_Interleaved(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 1000, TileSize: 100},
		{Name: "y", Size: 1000, TileSize: 100},
	}
	channels := ChannelSet{
		{Name: "elevation", Type: ChannelFloat32},
		{Name: "temperature", Type: ChannelFloat32},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, false)
	coord := SampleCoordinate{500, 500}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := SampleAt(memLayer, coord)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkChannelAt_LargeDimensions_2D_Interleaved(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 1000, TileSize: 100},
		{Name: "y", Size: 1000, TileSize: 100},
	}
	channels := ChannelSet{
		{Name: "elevation", Type: ChannelFloat32},
		{Name: "temperature", Type: ChannelFloat32},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, false)
	coord := SampleCoordinate{500, 500}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ChannelAt(memLayer, coord, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark with 3D dimensions
func BenchmarkSampleAt_3D_Interleaved(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 50, TileSize: 10},
		{Name: "y", Size: 50, TileSize: 10},
		{Name: "z", Size: 50, TileSize: 10},
	}
	channels := ChannelSet{
		{Name: "density", Type: ChannelFloat64},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, false)
	coord := SampleCoordinate{25, 25, 25}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := SampleAt(memLayer, coord)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkChannelAt_3D_Interleaved(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 50, TileSize: 10},
		{Name: "y", Size: 50, TileSize: 10},
		{Name: "z", Size: 50, TileSize: 10},
	}
	channels := ChannelSet{
		{Name: "density", Type: ChannelFloat64},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, false)
	coord := SampleCoordinate{25, 25, 25}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ChannelAt(memLayer, coord, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark with mixed channel types
func BenchmarkSampleAt_MixedTypes_Interleaved(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 100, TileSize: 10},
		{Name: "y", Size: 100, TileSize: 10},
	}
	channels := ChannelSet{
		{Name: "id", Type: ChannelUint32},
		{Name: "value", Type: ChannelFloat64},
		{Name: "flag", Type: ChannelUint8},
		{Name: "precision", Type: ChannelFloat32},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, false)
	coord := SampleCoordinate{50, 50}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := SampleAt(memLayer, coord)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkChannelAt_MixedTypes_Interleaved(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 100, TileSize: 10},
		{Name: "y", Size: 100, TileSize: 10},
	}
	channels := ChannelSet{
		{Name: "id", Type: ChannelUint32},
		{Name: "value", Type: ChannelFloat64},
		{Name: "flag", Type: ChannelUint8},
		{Name: "precision", Type: ChannelFloat32},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, false)
	coord := SampleCoordinate{50, 50}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Access the float64 channel (largest type) to test performance
		_, err := ChannelAt(memLayer, coord, 1)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark comparison for accessing multiple channels
// This simulates getting all channels individually vs getting them all at once
func BenchmarkChannelAt_AllChannels_Individual(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 100, TileSize: 10},
		{Name: "y", Size: 100, TileSize: 10},
	}
	channels := ChannelSet{
		{Name: "red", Type: ChannelUint8},
		{Name: "green", Type: ChannelUint8},
		{Name: "blue", Type: ChannelUint8},
		{Name: "alpha", Type: ChannelUint8},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, false)
	coord := SampleCoordinate{50, 50}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < len(channels); j++ {
			_, err := ChannelAt(memLayer, coord, j)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkSampleAt_AllChannels_AtOnce(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 100, TileSize: 10},
		{Name: "y", Size: 100, TileSize: 10},
	}
	channels := ChannelSet{
		{Name: "red", Type: ChannelUint8},
		{Name: "green", Type: ChannelUint8},
		{Name: "blue", Type: ChannelUint8},
		{Name: "alpha", Type: ChannelUint8},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, false)
	coord := SampleCoordinate{50, 50}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := SampleAt(memLayer, coord)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmarks for the zero-allocation SampleInto function
func BenchmarkSampleInto_SingleChannel_Interleaved(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 100, TileSize: 10},
		{Name: "y", Size: 100, TileSize: 10},
	}
	channels := ChannelSet{
		{Name: "value", Type: ChannelFloat32},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, false)
	coord := SampleCoordinate{50, 50}
	sample := make(Sample, len(channels)) // Pre-allocate reusable slice

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := SampleInto(memLayer, coord, sample)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSampleInto_MultiChannel_Interleaved(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 100, TileSize: 10},
		{Name: "y", Size: 100, TileSize: 10},
	}
	channels := ChannelSet{
		{Name: "red", Type: ChannelUint8},
		{Name: "green", Type: ChannelUint8},
		{Name: "blue", Type: ChannelUint8},
		{Name: "alpha", Type: ChannelUint8},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, false)
	coord := SampleCoordinate{50, 50}
	sample := make(Sample, len(channels)) // Pre-allocate reusable slice

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := SampleInto(memLayer, coord, sample)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSampleInto_MultiChannel_Separated(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 100, TileSize: 10},
		{Name: "y", Size: 100, TileSize: 10},
	}
	channels := ChannelSet{
		{Name: "red", Type: ChannelUint8},
		{Name: "green", Type: ChannelUint8},
		{Name: "blue", Type: ChannelUint8},
		{Name: "alpha", Type: ChannelUint8},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, true)
	coord := SampleCoordinate{50, 50}
	sample := make(Sample, len(channels)) // Pre-allocate reusable slice

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := SampleInto(memLayer, coord, sample)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSampleInto_MixedTypes_Interleaved(b *testing.B) {
	dims := DimensionSet{
		{Name: "x", Size: 100, TileSize: 10},
		{Name: "y", Size: 100, TileSize: 10},
	}
	channels := ChannelSet{
		{Name: "id", Type: ChannelUint32},
		{Name: "value", Type: ChannelFloat64},
		{Name: "flag", Type: ChannelUint8},
		{Name: "precision", Type: ChannelFloat32},
	}

	memLayer := setupBenchmarkLayer(b, dims, channels, false)
	coord := SampleCoordinate{50, 50}
	sample := make(Sample, len(channels)) // Pre-allocate reusable slice

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := SampleInto(memLayer, coord, sample)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestSampleInto_Functionality(t *testing.T) {
	header := Header{
		Version:    Version,
		OffsetSize: 4,
		ByteOrder:  binary.LittleEndian,
	}

	layer := NewLayer("test",
		DimensionSet{{Name: "x", Size: 10, TileSize: 5}, {Name: "y", Size: 10, TileSize: 5}},
		ChannelSet{{Name: "red", Type: ChannelUint8}, {Name: "green", Type: ChannelUint8}, {Name: "blue", Type: ChannelUint8}},
	)

	buf := buffer.NewBuffer(1000)
	memLayer := NewMemoryLayer(buf, header, layer)

	// Set some test data
	testCoord := SampleCoordinate{2, 3}
	testSample := Sample{uint8(255), uint8(128), uint8(64)}
	err := SetSampleAt(memLayer, testCoord, testSample)
	if err != nil {
		t.Fatal(err)
	}

	// Test SampleInto with correctly sized slice
	result := make(Sample, len(layer.Channels))
	err = SampleInto(memLayer, testCoord, result)
	if err != nil {
		t.Fatal(err)
	}
	if result[0] != uint8(255) || result[1] != uint8(128) || result[2] != uint8(64) {
		t.Fatalf("expected [255 128 64], got %v", result)
	}

	// Test that SampleAt and SampleInto return the same values
	originalSample, err := SampleAt(memLayer, testCoord)
	if err != nil {
		t.Fatal(err)
	}

	reusableSample := make(Sample, len(layer.Channels))
	err = SampleInto(memLayer, testCoord, reusableSample)
	if err != nil {
		t.Fatal(err)
	}

	for i := range originalSample {
		if originalSample[i] != reusableSample[i] {
			t.Fatalf("channel %d mismatch: SampleAt returned %v, SampleInto returned %v", i, originalSample[i], reusableSample[i])
		}
	}
}

func TestSampleInto_ReuseSlice(t *testing.T) {
	header := Header{
		Version:    Version,
		OffsetSize: 4,
		ByteOrder:  binary.LittleEndian,
	}

	layer := NewLayer("test",
		DimensionSet{{Name: "x", Size: 10, TileSize: 5}},
		ChannelSet{{Name: "value", Type: ChannelFloat32}},
	)

	buf := buffer.NewBuffer(1000)
	memLayer := NewMemoryLayer(buf, header, layer)

	// Set test data at different coordinates
	coords := []SampleCoordinate{{1}, {2}, {3}}
	values := []float32{1.1, 2.2, 3.3}

	for i, coord := range coords {
		err := SetSampleAt(memLayer, coord, Sample{values[i]})
		if err != nil {
			t.Fatal(err)
		}
	}

	// Reuse the same slice for all reads
	reusableSlice := make(Sample, 1)
	for i, coord := range coords {
		err := SampleInto(memLayer, coord, reusableSlice)
		if err != nil {
			t.Fatal(err)
		}
		if reusableSlice[0] != values[i] {
			t.Fatalf("coord %v: expected %f, got %v", coord, values[i], reusableSlice[0])
		}
	}
}
