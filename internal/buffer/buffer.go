package buffer

import (
	"io"
)

type buffer struct {
	buf []byte
	pos int
	end int
}

func NewBuffer(initialSize int) *buffer {
	return &buffer{
		buf: make([]byte, initialSize),
	}
}

func NewBufferFrom(underlying []byte) *buffer {
	return &buffer{
		buf: underlying,
		end: len(underlying),
	}
}

func (b *buffer) Read(p []byte) (int, error) {
	if len(p) > 0 && b.pos < len(b.buf) {
		n := copy(p, b.buf[b.pos:])
		b.pos += n
		return n, nil
	} else if b.pos >= len(b.buf) {
		return 0, io.EOF
	} else if len(p) == 0 {
		return 0, nil
	} else {
		return 0, io.ErrUnexpectedEOF
	}
}

func (b *buffer) Write(p []byte) (int, error) {
	for b.pos+len(p) >= len(b.buf) {
		b.buf = append(b.buf, make([]byte, len(b.buf))...)
	}
	n := copy(b.buf[b.pos:], p)
	b.pos += n
	b.end = max(b.end, b.pos)
	return n, nil
}

func (b *buffer) Seek(offset int64, whence int) (int64, error) {
	var newOffset int
	switch whence {
	case io.SeekStart:
		newOffset = int(offset)
	case io.SeekCurrent:
		if offset >= 0 {
			newOffset = b.pos + int(offset)
		} else if offset < 0 {
			newOffset = max(0, b.pos+int(offset))
		}
	case io.SeekEnd:
		newOffset = len(b.buf) + int(offset)
	default:
		panic("pixi: invalid whence in buffer seek")
	}

	if newOffset > len(b.buf) {
		return -1, io.ErrUnexpectedEOF
	}

	b.pos = newOffset

	return int64(b.pos), nil
}

func (b *buffer) Bytes() []byte {
	return b.buf[:b.end]
}

func (b *buffer) Size() int {
	return len(b.buf)
}

func (b *buffer) Position() int {
	return b.pos
}
