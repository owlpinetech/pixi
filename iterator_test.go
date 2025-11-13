package pixi

import (
	"encoding/binary"
	"testing"

	"github.com/owlpinetech/pixi/internal/buffer"
)

func TestTileOrderReadIterator(t *testing.T) {
	header := &PixiHeader{
		Version:    Version,
		OffsetSize: 4,
		ByteOrder:  binary.BigEndian,
	}

	// write some test data
	wrtBuf := buffer.NewBuffer(10)
	layer, err := NewBlankUncompressedLayer(
		wrtBuf,
		header,
		"tile-order-read-iterator-test",
		false,
		DimensionSet{{Name: "x", Size: 500, TileSize: 100}, {Name: "y", Size: 500, TileSize: 100}},
		FieldSet{{Name: "one", Type: FieldUint16}, {Name: "two", Type: FieldUint32}},
	)
	if err != nil {
		t.Fatal(err)
	}

	stored := NewStoredLayer(wrtBuf, header, layer)
	stored.SetFieldAt(SampleCoordinate{0, 0}, 0, uint16(123))
	stored.SetFieldAt(SampleCoordinate{0, 0}, 1, uint32(456789))
	stored.SetFieldAt(SampleCoordinate{499, 499}, 0, uint16(321))
	stored.SetFieldAt(SampleCoordinate{499, 499}, 1, uint32(987654))
	stored.SetFieldAt(SampleCoordinate{250, 250}, 0, uint16(111))
	stored.SetFieldAt(SampleCoordinate{250, 250}, 1, uint32(222222))

	rdBuffer := buffer.NewBufferFrom(wrtBuf.Bytes())
	iterator := NewTileOrderReadIterator(rdBuffer, header, layer)
	lastTileIndex := TileOrderIndex(-1)
	defer iterator.Done()
	for iterator.Next() {
		coord := iterator.Coordinate()

		// monotonically increasing tile order index
		tileOrderIndex := coord.ToTileCoordinate(layer.Dimensions).ToTileSelector(layer.Dimensions).ToTileIndex(layer.Dimensions)
		if tileOrderIndex <= lastTileIndex {
			t.Errorf("Tile order iterator returned samples out of order: last index %d, current index %d", lastTileIndex, tileOrderIndex)
		}

		// multiple accesss to the same sample should return the same result
		sample := iterator.Sample()
		sampleAgain := iterator.Sample()
		for i := range sample {
			if sample[i] != sampleAgain[i] {
				t.Errorf("Tile order iterator returned different results for multiple accesses to the same sample at index %d: first %v, second %v", tileOrderIndex, sample, sampleAgain)
			}
		}

		for fieldIndex := range layer.Fields {
			if len((sample)) != len(layer.Fields) {
				t.Errorf("Tile order iterator Sample() length does not match field count at index %d: got %d, expected %d", tileOrderIndex, len(sample), len(layer.Fields))
			}
			fieldValue := iterator.Field(fieldIndex)
			if fieldValue != sample[fieldIndex] {
				t.Errorf("Tile order iterator Field() result does not match Sample() result at index %d, field %d: Field() %v, Sample() %v", tileOrderIndex, fieldIndex, fieldValue, sample[fieldIndex])
			}

			// compare against raw data
			expectedValue, err := stored.FieldAt(coord, fieldIndex)
			if err != nil {
				t.Errorf("Error retrieving sample at coord %v for comparison: %v", coord, err)
			}
			if fieldValue != expectedValue {
				t.Errorf("Tile order iterator returned incorrect value at index %d, field %d: got %v, expected %v", tileOrderIndex, fieldIndex, fieldValue, expectedValue)
			}
		}
	}
}

func TestTileOrderWriteIterator(t *testing.T) {
	header := &PixiHeader{
		Version:    Version,
		OffsetSize: 4,
		ByteOrder:  binary.BigEndian,
	}
	layer := NewLayer(
		"tile-order-write-iterator-test",
		false,
		CompressionNone,
		DimensionSet{{Name: "x", Size: 500, TileSize: 100}, {Name: "y", Size: 500, TileSize: 100}},
		FieldSet{{Name: "one", Type: FieldUint16}, {Name: "two", Type: FieldUint32}})

	wrtBuf := buffer.NewBuffer(10)

	iterator := NewTileOrderWriteIterator(wrtBuf, header, layer)
	defer iterator.Done()

	for iterator.Next() {
		coord := iterator.Coordinate()
		sample := make(Sample, len(layer.Fields))
		if coord[0] == 0 && coord[1] == 0 {
			sample[0] = uint16(123)
			sample[1] = uint32(456789)
		} else if coord[0] == 499 && coord[1] == 499 {
			sample[0] = uint16(321)
			sample[1] = uint32(987654)
		} else if coord[0] == 250 && coord[1] == 250 {
			sample[0] = uint16(111)
			sample[1] = uint32(222222)
		} else {
			sample[0] = uint16(0)
			sample[1] = uint32(0)
		}
		iterator.SetSample(sample)
	}

	if iterator.Error() != nil {
		t.Fatalf("Tile order write iterator encountered error: %v", iterator.Error())
	}

	rdBuffer := buffer.NewBufferFrom(wrtBuf.Bytes())
	stored := NewStoredLayer(rdBuffer, header, layer)

	checks := []struct {
		coord  SampleCoordinate
		expect Sample
	}{
		{SampleCoordinate{0, 0}, Sample{uint16(123), uint32(456789)}},
		{SampleCoordinate{499, 499}, Sample{uint16(321), uint32(987654)}},
		{SampleCoordinate{250, 250}, Sample{uint16(111), uint32(222222)}},
	}

	for _, check := range checks {
		sample, err := stored.SampleAt(check.coord)
		if err != nil {
			t.Errorf("Error retrieving sample at coord %v: %v", check.coord, err)
			continue
		}
		for i := range check.expect {
			if sample[i] != check.expect[i] {
				t.Errorf("Incorrect value at coord %v, field %d: got %v, expected %v", check.coord, i, sample[i], check.expect[i])
			}
		}
	}
}
