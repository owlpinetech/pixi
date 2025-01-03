package pixi

import (
	"bytes"
	"compress/flate"
	"io"
)

// Represents the compression method used to shrink the data persisted to a layer in a Pixi file.
type Compression uint32

const (
	CompressionNone  Compression = 0 // No compression
	CompressionFlate Compression = 1 // Standard FLATE compression
)

func (c Compression) String() string {
	switch c {
	case CompressionNone:
		return "none"
	default:
		return "flate"
	}
}

// Compresses the given chunk of data according to the selected compression scheme, and writes
// the compressed data to the writer. Returns the number of compressed bytes written, or an error
// if the write failed.
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

// Reads a compressed chunk of data into the given slice which must be the size of the desired
// uncompressed data. Returns the number of bytes read or and error if the read failed.
func (c Compression) ReadChunk(r io.Reader, chunk []byte) (int, error) {
	switch c {
	case CompressionNone:
		return r.Read(chunk)
	case CompressionFlate:
		byteBuf := make([]byte, 0, len(chunk))
		bufRd := bytes.NewBuffer(byteBuf)
		gzRdr := flate.NewReader(bufRd)
		defer gzRdr.Close()
		amtRd, err := io.Copy(bufRd, gzRdr)
		copy(chunk, bufRd.Bytes())
		return int(amtRd), err
	default:
		return 0, UnsupportedError("unknown compression")
	}
}
