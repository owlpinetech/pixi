package gopixi

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
)

const (
	offsetsOffset int64 = 8

	OffsetSize4 OffsetSize = 4
	OffsetSize8 OffsetSize = 8
)

type OffsetSize int

// Contains information used to read or write the rest of a Pixi data file. This information
// is always found at the start of a stream of Pixi data. Because so much of how the rest of
// the file is serialized is dependent on this information, it is threaded throughout the
// reading and writing methods of the other structures that make up a Pixi stream.
type Header struct {
	Version          int
	OffsetSize       OffsetSize
	ByteOrder        binary.ByteOrder
	FirstLayerOffset int64
	FirstTagsOffset  int64
}

// Creates a new Pixi header struct with the given byte order and offset size, setting
// the version to the current supported version.
func NewHeader(byteOrder binary.ByteOrder, offsetSize OffsetSize) Header {
	return Header{
		Version:    Version,
		OffsetSize: offsetSize,
		ByteOrder:  byteOrder,
	}
}

// Get the size in bytes of the full Pixi header (including first tag section and first layer offsets) as it is laid out and written to disk.
func (s Header) DiskSize() int {
	return 4 + 2 + 1 + 1 + 2*int(s.OffsetSize)
}

// Writes a fixed size value, or a slice of such values, using the byte order given in the header.
func (s Header) Write(w io.Writer, val any) error {
	return binary.Write(w, s.ByteOrder, val)
}

// Reads a fixed-size value, or a slice of such values, using the byte order given in the header.
func (s Header) Read(r io.Reader, val any) error {
	return binary.Read(r, s.ByteOrder, val)
}

// Writes a file offset to the current position in the writer stream, based on the offset size
// specified in the header. Panics if the file offset size has not yet been set, and returns
// an error if writing fails.
func (s Header) WriteOffset(w io.Writer, offset int64) error {
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
func (s Header) ReadOffset(r io.Reader) (int64, error) {
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
func (s Header) WriteOffsets(w io.Writer, offsets []int64) error {
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
func (s Header) ReadOffsets(r io.Reader, offsets []int64) error {
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

// Writes a 'friendly' name from to the writer stream at the current position. A
// 'friendly' string is always the same format, specified by a 16-bit length followed
// by that number of bytes of UTF8 string.
func (s Header) WriteFriendly(w io.Writer, friendly string) error {
	strBytes := []byte(friendly)
	err := s.Write(w, uint16(len(strBytes)))
	if err != nil {
		return nil
	}
	return s.Write(w, strBytes)
}

// Read a 'friendly' name from the reader stream at the current position. 'Friendly'
// strings are always the same format, specified by a 16-bit length followed by that
// number of bytes interpreted as a UTF8 string.
func (s Header) ReadFriendly(r io.Reader) (string, error) {
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
func (h Header) WriteHeader(w io.Writer) error {
	// write file type (4 bytes)
	_, err := w.Write([]byte(FileType))
	if err != nil {
		return err
	}

	// write file version (2 bytes)
	_, err = fmt.Fprintf(w, "%02d", h.Version)
	if err != nil {
		return err
	}

	// write offset size indicator (1 byte)
	_, err = w.Write([]byte{byte(h.OffsetSize)})
	if err != nil {
		return err
	}

	// write byte order indicator (1 byte)
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
func (h *Header) ReadHeader(r io.Reader) error {
	buf := make([]byte, 4)

	// check file type
	_, err := r.Read(buf)
	if err != nil {
		return err
	}
	fileType := string(buf)
	if fileType != FileType {
		return ErrFormat("pixi file marker not found at start of file")
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
	if int(version) > Version {
		return ErrFormat("reader does not support this version of pixi file")
	}

	h.Version = int(version)

	// read offset size indicator & byte order indicator
	_, err = r.Read(buf[0:2])
	if err != nil {
		return err
	}

	if buf[0] != 4 && buf[0] != 8 {
		return ErrFormat("reader only supports offset sizes of 4 or 8 bytes")
	}
	h.OffsetSize = OffsetSize(buf[0])

	switch buf[1] {
	case 0x00:
		h.ByteOrder = binary.LittleEndian
	case 0xff:
		h.ByteOrder = binary.BigEndian
	default:
		return ErrFormat("unsupported or invalid byte order specified")
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

// Temporarily returns to the beginning of the Pixi stream to overwrite the offsets of the first layer
// and the first tags sections. Useful during initial file creation or editing, especially for large data
// that is difficult to know the size of in advance. After completing the offsets overwrite, or upon encountering
// an error in attempting to do so, this function will return the cursor to the position at which it was
// when the call to this function was made.
func (h *Header) OverwriteOffsets(w io.WriteSeeker, firstLayer int64, firstTags int64) error {
	oldPos, err := w.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	defer w.Seek(oldPos, io.SeekStart)
	_, err = w.Seek(offsetsOffset, io.SeekStart)
	if err != nil {
		return err
	}

	err = h.WriteOffset(w, firstLayer)
	if err != nil {
		return err
	}
	h.FirstLayerOffset = firstLayer

	err = h.WriteOffset(w, firstTags)
	if err != nil {
		return err
	}
	h.FirstTagsOffset = firstTags

	return nil
}

func allHeaderVariants(version int) []Header {
	return []Header{
		{Version: version, ByteOrder: binary.BigEndian, OffsetSize: 4},
		{Version: version, ByteOrder: binary.BigEndian, OffsetSize: 8},
		{Version: version, ByteOrder: binary.LittleEndian, OffsetSize: 4},
		{Version: version, ByteOrder: binary.LittleEndian, OffsetSize: 8},
	}
}
