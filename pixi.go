package pixi

import (
	"io"
	"slices"
)

const (
	FileType string = "pixi" // Every file starts with these four bytes.
	Version  int    = 1      // Every file has a version number as the second set of four bytes.
)

// Represents a single pixi file composed of one or more layers. Functions as a handle
// to access the description of the each layer as well as the data stored in each layer.
type Pixi struct {
	Header *PixiHeader   // The metadata about the file version and how to read information from the file.
	Layers []*Layer      // The metadata information about each layer in the file.
	Tags   []*TagSection // The string tags of the file, broken up into sections for easy appending.
}

// Convenience function to read all the metadata information from a Pixi file into a single
// containing struct.
func ReadPixi(r io.ReadSeeker) (*Pixi, error) {
	pixi := &Pixi{
		Header: &PixiHeader{},
		Layers: make([]*Layer, 0),
		Tags:   make([]*TagSection, 0),
	}

	seenOffsets := []int64{}

	// read the header first, then the layers and tags.
	err := pixi.Header.ReadHeader(r)
	if err != nil {
		return pixi, err
	}

	layerOffset := pixi.Header.FirstLayerOffset
	for layerOffset != 0 {
		if slices.Contains(seenOffsets, layerOffset) {
			return pixi, FormatError("loop detected in layer offsets")
		}
		seenOffsets = append(seenOffsets, layerOffset)
		_, err = r.Seek(layerOffset, io.SeekStart)
		if err != nil {
			return pixi, err
		}
		rdLayer := &Layer{}
		err = rdLayer.ReadLayer(r, pixi.Header)
		if err != nil {
			return pixi, err
		}
		pixi.Layers = append(pixi.Layers, rdLayer)
		layerOffset = rdLayer.NextLayerStart
	}

	tagOffset := pixi.Header.FirstTagsOffset
	for tagOffset != 0 {
		if slices.Contains(seenOffsets, tagOffset) {
			return pixi, FormatError("loop detected in tag offsets")
		}
		seenOffsets = append(seenOffsets, layerOffset)
		_, err := r.Seek(tagOffset, io.SeekStart)
		if err != nil {
			return pixi, err
		}
		rdTags := &TagSection{}
		err = rdTags.Read(r, pixi.Header)
		if err != nil {
			return pixi, err
		}
		pixi.Tags = append(pixi.Tags, rdTags)
		tagOffset = rdTags.NextTagsStart
	}

	return pixi, nil
}

// The total size of the data portions of the file in bytes. Does not count header information
// as part of the size.
func (d *Pixi) DiskDataBytes() int64 {
	size := int64(0)
	for _, l := range d.Layers {
		for _, t := range l.TileBytes {
			size += t
		}
	}
	return size
}
