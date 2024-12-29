package pixi

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
)

// Contains information used to read or write the rest of a Pixi data file. This information
// is always found at the start of a stream of Pixi data.
type PixiHeader struct {
	Version          int
	OffsetSize       int
	ByteOrder        binary.ByteOrder
	FirstLayerOffset int64
	FirstTagsOffset  int64
}

// Writes a fixed size value, or a slice of such values, using the byte order given in the header.
func (s *PixiHeader) Write(w io.Writer, val any) error {
	return binary.Write(w, s.ByteOrder, val)
}

// Reads a fixed-size value, or a slice of such values, using the byte order given in the header.
func (s *PixiHeader) Read(r io.Reader, val any) error {
	return binary.Read(r, s.ByteOrder, val)
}

// Writes a file offset to the current position in the writer stream, based on the offset size
// specified in the header. Panics if the file offset size has not yet been set, and returns
// an error if writing fails.
func (s *PixiHeader) WriteOffset(w io.Writer, offset int64) error {
	switch s.OffsetSize {
	case 4:
		return binary.Write(w, s.ByteOrder, int32(offset))
	case 8:
		return binary.Write(w, s.ByteOrder, offset)
	}
	panic("pixi: unsupported offset size")
}

// Reads a file offset from the current position in the reader, based on the offset size
// read earlier in the file. Panics if the file offset size has not yet been set, and returns
// an error if reading fails.
func (s *PixiHeader) ReadOffset(r io.Reader) (int64, error) {
	switch s.OffsetSize {
	case 4:
		var offset int32
		err := binary.Read(r, s.ByteOrder, &offset)
		return int64(offset), err
	case 8:
		var offset int64
		err := binary.Read(r, s.ByteOrder, &offset)
		return offset, err
	}
	panic("pixi: unsupported offset size")
}

// Writes a slice of offsets to the current position in the writer stream, based on the offset size
// specified in the header. Panics if the file offset size has not yet been set, and returns
// an error if writing fails.
func (s *PixiHeader) WriteOffsets(w io.Writer, offsets []int64) error {
	switch s.OffsetSize {
	case 4:
		smallOffs := make([]int32, len(offsets))
		for i := range offsets {
			smallOffs[i] = int32(offsets[i])
		}
		return binary.Write(w, s.ByteOrder, smallOffs)
	case 8:
		return binary.Write(w, s.ByteOrder, offsets)
	}
	panic("pixi: unsupported offset size")
}

// Reads a slice of offsets from the current position in the reader, based on the offset size
// read earlier in the file. Panics if the file offset size has not yet been set, and returns
// an error if reading fails.
func (s *PixiHeader) ReadOffsets(r io.Reader, offsets []int64) error {
	switch s.OffsetSize {
	case 4:
		smallOffs := make([]int32, len(offsets))
		err := binary.Read(r, s.ByteOrder, smallOffs)
		if err != nil {
			return err
		}
		for i := range offsets {
			offsets[i] = int64(smallOffs[i])
		}
		return nil
	case 8:
		return binary.Read(r, s.ByteOrder, offsets)
	}
	panic("pixi: unsupported offset size")
}

func (s *PixiHeader) WriteFriendly(w io.Writer, friendly string) error {
	strBytes := []byte(friendly)
	err := s.Write(w, uint16(len(strBytes)))
	if err != nil {
		return nil
	}
	return s.Write(w, strBytes)
}

func (s *PixiHeader) ReadFriendly(r io.Reader) (string, error) {
	var strLen uint16
	err := s.Read(r, &strLen)
	if err != nil {
		return "", err
	}
	strBytes := make([]byte, int(strLen))
	err = s.Read(r, strBytes)
	return string(strBytes), err
}

// Write the information in this header to the current position in the writer stream.
func (h *PixiHeader) WriteHeader(w io.Writer) error {
	// write file type
	_, err := w.Write([]byte(FileType))
	if err != nil {
		return err
	}

	// write file version
	_, err = w.Write([]byte(fmt.Sprintf("%02d", h.Version)))
	if err != nil {
		return err
	}

	// write offset size indicator
	_, err = w.Write([]byte{byte(h.OffsetSize)})
	if err != nil {
		return err
	}

	// write byte order indicator
	byteOrderEnc := byte(0x00)
	if h.ByteOrder == binary.BigEndian {
		byteOrderEnc = byte(0xff)
	}
	_, err = w.Write([]byte{byteOrderEnc})
	if err != nil {
		return err
	}

	// write first layer offset
	err = h.WriteOffset(w, h.FirstLayerOffset)
	if err != nil {
		return err
	}

	// write first tags offset
	return h.WriteOffset(w, h.FirstTagsOffset)
}

// Read Pixi header information into this struct from the current position in the reader stream.
// Will return an error if the reading fails, or if there are format errors in the Pixi header.
func (h *PixiHeader) ReadHeader(r io.Reader) error {
	buf := make([]byte, 4)

	// check file type
	_, err := r.Read(buf)
	if err != nil {
		return err
	}
	fileType := string(buf)
	if fileType != FileType {
		return FormatError("pixi file marker not found at start of file")
	}

	// check file version
	_, err = r.Read(buf[0:2])
	if err != nil {
		return err
	}
	version, err := strconv.ParseInt(string(buf[0:2]), 10, 32)
	if err != nil {
		return err
	}
	if version > Version {
		return FormatError("reader does not support this version of pixi file")
	}

	h.Version = int(version)

	// read offset size indicator & byte order indicator
	_, err = r.Read(buf[0:2])
	if err != nil {
		return err
	}

	if buf[0] != 4 && buf[0] != 8 {
		return FormatError("reader only supports offset sizes of 4 or 8 bytes")
	}
	h.OffsetSize = int(buf[0])

	if buf[1] == 0x00 {
		h.ByteOrder = binary.LittleEndian
	} else if buf[1] == 0xff {
		h.ByteOrder = binary.BigEndian
	} else {
		return FormatError("unsupported or invalid byte order specified")
	}

	// read first layer offset
	firstLayerOffset, err := h.ReadOffset(r)
	if err != nil {
		return err
	}
	h.FirstLayerOffset = firstLayerOffset

	// read tagging section offset
	firstTagsOffset, err := h.ReadOffset(r)
	if err != nil {
		return err
	}
	h.FirstTagsOffset = firstTagsOffset

	return nil
}
