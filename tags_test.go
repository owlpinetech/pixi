package gopixi

import (
	"encoding/binary"
	"testing"

	"github.com/gracefulearth/gopixi/internal/buffer"
)

func TestTagSectionWriteRead(t *testing.T) {
	header := Header{
		Version:    Version,
		OffsetSize: 4,
		ByteOrder:  binary.BigEndian,
	}

	// write some test tags
	wrtBuf := buffer.NewBuffer(10)
	tags := TagSection{
		Tags: map[string]string{
			"author":      "testuser",
			"description": "this is a test image",
		},
	}
	err := tags.Write(wrtBuf, header)
	if err != nil {
		t.Fatal(err)
	}

	// read back the tags
	rdBuffer := buffer.NewBufferFrom(wrtBuf.Bytes())
	readTags := &TagSection{}
	err = readTags.Read(rdBuffer, header)
	if err != nil {
		t.Fatal(err)
	}

	// compare tags
	for key, expectedValue := range tags.Tags {
		readValue, exists := readTags.Tags[key]
		if !exists {
			t.Errorf("tag %s missing in read tags", key)
			continue
		}
		if readValue != expectedValue {
			t.Errorf("tag %s value mismatch: expected %v, got %v", key, expectedValue, readValue)
		}
	}
}
