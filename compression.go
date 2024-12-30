package pixi

import (
	"bytes"
	"compress/flate"
	"io"
)

type Compression uint32

const (
	CompressionNone  Compression = 0
	CompressionFlate Compression = 1
)

func (c Compression) String() string {
	switch c {
	case CompressionNone:
		return "none"
	default:
		return "flate"
	}
}

func (c Compression) WriteChunk(w io.Writer, chunk []byte) (int, error) {
	switch c {
	case CompressionNone:
		return w.Write(chunk)
	case CompressionFlate:
		// we have to write to a buffer so we can get the actual amount the compression writes
		buf := new(bytes.Buffer)
		flateWriter, err := flate.NewWriter(buf, flate.BestCompression)
		if err != nil {
			return 0, err
		}
		// skip this amount; it just returns len(chunk)!
		_, err = flateWriter.Write(chunk)
		if err != nil {
			flateWriter.Close()
			return 0, err
		}
		flateWriter.Close()
		writeAmt, err := io.Copy(w, buf)
		return int(writeAmt), err
	default:
		return 0, UnsupportedError("unknown compression")
	}
}

func (c Compression) ReadChunk(r io.Reader, chunk []byte) (int, error) {
	switch c {
	case CompressionNone:
		return r.Read(chunk)
	case CompressionFlate:
		bufRd := bytes.NewBuffer(chunk)
		gzRdr := flate.NewReader(bufRd)
		defer gzRdr.Close()
		amtRd, err := io.Copy(bufRd, gzRdr)
		return int(amtRd), err
	default:
		return 0, UnsupportedError("unknown compression")
	}
}
