package gopixi

import (
	"fmt"
	"io"
	"maps"
	"slices"
)

const (
	FileType string = "pixi" // Every file starts with these four bytes.
	Version  int    = 1      // Every file has a version number as the second set of four bytes.
)

// Represents a single pixi file composed of one or more layers. Functions as a handle
// to access the description of the each layer as well as the data stored in each layer.
type Pixi struct {
	Header Header       // The metadata about the file version and how to read information from the file.
	Layers []Layer      // The metadata information about each layer in the file.
	Tags   []TagSection // The string tags of the file, broken up into sections for easy appending.
}

// Convenience function to read all the metadata information from a Pixi file into a single
// containing struct.
func ReadPixi(r io.ReadSeeker) (*Pixi, error) {
	pixi := &Pixi{
		Header: Header{},
		Layers: make([]Layer, 0),
		Tags:   make([]TagSection, 0),
	}

	seenOffsets := []int64{}

	// read the header first, then the layers and tags.
	err := pixi.Header.ReadHeader(r)
	if err != nil {
		return pixi, ErrFormat(fmt.Sprintf("reading pixi header: %s", err))
	}

	layerOffset := pixi.Header.FirstLayerOffset
	for layerOffset != 0 {
		if slices.Contains(seenOffsets, layerOffset) {
			return pixi, ErrFormat("loop detected in layer offsets")
		}
		seenOffsets = append(seenOffsets, layerOffset)
		_, err = r.Seek(layerOffset, io.SeekStart)
		if err != nil {
			return pixi, ErrFormat(fmt.Sprintf("seeking to layer at offset %d: %s", layerOffset, err))
		}
		rdLayer := Layer{}
		err = rdLayer.ReadLayer(r, pixi.Header)
		if err != nil {
			return pixi, ErrFormat(fmt.Sprintf("reading layer at offset %d: %s", layerOffset, err))
		}
		pixi.Layers = append(pixi.Layers, rdLayer)
		layerOffset = rdLayer.NextLayerStart
	}

	tagOffset := pixi.Header.FirstTagsOffset
	for tagOffset != 0 {
		if slices.Contains(seenOffsets, tagOffset) {
			return pixi, ErrFormat("loop detected in tag offsets")
		}
		seenOffsets = append(seenOffsets, layerOffset)
		_, err := r.Seek(tagOffset, io.SeekStart)
		if err != nil {
			return pixi, ErrFormat(fmt.Sprintf("seeking to tag section at offset %d: %s", tagOffset, err))
		}
		rdTags := TagSection{}
		err = rdTags.Read(r, pixi.Header)
		if err != nil {
			return pixi, ErrFormat(fmt.Sprintf("reading tag section at offset %d: %s", tagOffset, err))
		}
		pixi.Tags = append(pixi.Tags, rdTags)
		tagOffset = rdTags.NextTagsStart
	}

	return pixi, nil
}

func (d *Pixi) AllTags() map[string]string {
	tags := map[string]string{}
	for _, t := range d.Tags {
		maps.Copy(tags, t.Tags)
	}
	return tags
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

// Appends a new tag section to the end of the file with the given tags.
func (p *Pixi) AppendTags(w io.WriteSeeker, tags map[string]string) error {
	// Append the new tag section to the end of the file
	tagSectionStart, err := w.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	newTagSection := TagSection{
		Tags: tags,
	}
	err = newTagSection.Write(w, p.Header)
	if err != nil {
		return err
	}

	// Update the previous tag section (or the header if this is the first)
	if len(p.Tags) == 0 {
		if err = p.Header.OverwriteOffsets(w, p.Header.FirstLayerOffset, tagSectionStart); err != nil {
			return err
		}
	} else {
		var prevTagOffset int64
		if len(p.Tags) == 1 {
			prevTagOffset = p.Header.FirstTagsOffset
		} else {
			prevTagOffset = p.Tags[len(p.Tags)-2].NextTagsStart
		}
		_, err = w.Seek(prevTagOffset, io.SeekStart)
		if err != nil {
			return err
		}

		p.Tags[len(p.Tags)-1].NextTagsStart = tagSectionStart
		err = p.Tags[len(p.Tags)-1].WriteHeader(w, p.Header)
		if err != nil {
			return err
		}
	}

	p.Tags = append(p.Tags, newTagSection)
	return nil
}

// Appends a new layer to the end of the file, using the provided generator function for writing samples to the layer.
func (p *Pixi) AppendIterativeLayer(w io.WriteSeeker, layer Layer, writer IterativeLayerWriter, generator func(writer IterativeLayerWriter) error) error {
	// append the new layer to the end of the file
	_, err := w.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	// write out all the tile data
	if err := generator(writer); err != nil {
		return err
	}
	writer.Done()
	if err := writer.Error(); err != nil {
		return err
	}

	// write out the layer metadata
	layerStart, err := w.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	err = layer.WriteHeader(w, p.Header)
	if err != nil {
		return err
	}

	// update the previous layer (or the header if this is the first)
	if len(p.Layers) == 0 {
		if err = p.Header.OverwriteOffsets(w, layerStart, p.Header.FirstTagsOffset); err != nil {
			return err
		}
	} else {
		var prevLayerOffset int64
		if len(p.Layers) == 1 {
			prevLayerOffset = p.Header.FirstLayerOffset
		} else {
			prevLayerOffset = p.Layers[len(p.Layers)-2].NextLayerStart
		}
		_, err = w.Seek(prevLayerOffset, io.SeekStart)
		if err != nil {
			return err
		}

		p.Layers[len(p.Layers)-1].NextLayerStart = layerStart
		err = p.Layers[len(p.Layers)-1].WriteHeader(w, p.Header)
		if err != nil {
			return err
		}
	}

	p.Layers = append(p.Layers, layer)
	return nil
}
