package pixi

import "io"

// Pixi files can contain zero or more tag sections, used for extraneous non-data related metadata
// to help describe the file or indicate context of the file's ownership and lifespan. While the tags
// are conceptually just a flat list of string pairs, the layout in the file is done in sections with
// offsets pointing to further sections, allowing easier 'appending' of additional tags regardless of
// where in the file previous tags are stored.
type TagSection struct {
	Tags          map[string]string // The tags for this section.
	NextTagsStart int64             // A byte-index offset from the start of the file pointing to the next tag section. 0 if this is the last tag section.
}

// Writes the tag section header in binary to the given stream, according to the specification
// in the Pixi header. This only writes the header (number of tags and offset to next section),
// not the actual tags themselves.
func (t *TagSection) WriteHeader(w io.Writer, h *Header) error {
	err := h.Write(w, uint32(len(t.Tags)))
	if err != nil {
		return err
	}
	return h.WriteOffset(w, t.NextTagsStart)
}

// Writes the tag section in binary to the given stream, according to the specification
// in the Pixi header.
func (t *TagSection) Write(w io.Writer, h *Header) error {
	// write number of tags, then each key-value pair for tags
	err := h.Write(w, uint32(len(t.Tags)))
	if err != nil {
		return err
	}
	for k, v := range t.Tags {
		err = h.WriteFriendly(w, k)
		if err != nil {
			return err
		}
		err = h.WriteFriendly(w, v)
		if err != nil {
			return err
		}
	}
	return nil
}

// Reads a tag section from the given binary stream, according to the specification
// in the Pixi header.
func (t *TagSection) Read(r io.Reader, h *Header) error {
	var tagCount uint32
	err := h.Read(r, &tagCount)
	if err != nil {
		return err
	}
	t.NextTagsStart, err = h.ReadOffset(r)
	if err != nil {
		return err
	}
	t.Tags = make(map[string]string)
	for range tagCount {
		key, err := h.ReadFriendly(r)
		if err != nil {
			return err
		}
		val, err := h.ReadFriendly(r)
		if err != nil {
			return err
		}
		t.Tags[key] = val
	}
	return nil
}
