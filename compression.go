package pixi

import (
	"bytes"
	"compress/flate"
	"compress/lzw"
	"io"
)

// Represents the compression method used to shrink the data persisted to a layer in a Pixi file.
type Compression uint32

const (
	CompressionNone   Compression = 0 // No compression
	CompressionFlate  Compression = 1 // Standard FLATE compression
	CompressionLzwLsb Compression = 2 // Least-significant-bit Lempel-Ziv-Welch compression from Go standard lib
	CompressionLzwMsb Compression = 3 // Most-significant-bit Lempel-Ziv-Welch compression from Go standard lib
	CompressionRle8   Compression = 4 // Run-length encoding capable of compressing up to 255 repeats of a sample
)

func (c Compression) String() string {
	switch c {
	case CompressionNone:
		return "none"
	case CompressionFlate:
		return "flate"
	case CompressionLzwLsb:
		return "lzw_lsb"
	case CompressionLzwMsb:
		return "lzw_msb"
	case CompressionRle8:
		return "rle"
	default:
		return "unknown"
	}
}

// Compresses the given chunk of data according to the selected compression scheme, and writes
// the compressed data to the writer. Returns the number of compressed bytes written, or an error
// if the write failed.
func (c Compression) writeChunk(w io.Writer, layer *Layer, tileIndex int, chunk []byte) (int, error) {
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
	case CompressionLzwLsb:
		// we have to write to a buffer so we can get the actual amount the compression writes
		buf := new(bytes.Buffer)
		lzwWriter := lzw.NewWriter(buf, lzw.LSB, 8)

		// skip this amount; it just returns len(chunk)!
		_, err := lzwWriter.Write(chunk)
		if err != nil {
			lzwWriter.Close()
			return 0, err
		}
		lzwWriter.Close()
		writeAmt, err := io.Copy(w, buf)
		return int(writeAmt), err
	case CompressionLzwMsb:
		// we have to write to a buffer so we can get the actual amount the compression writes
		buf := new(bytes.Buffer)
		lzwWriter := lzw.NewWriter(buf, lzw.MSB, 8)

		// skip this amount; it just returns len(chunk)!
		_, err := lzwWriter.Write(chunk)
		if err != nil {
			lzwWriter.Close()
			return 0, err
		}
		lzwWriter.Close()
		writeAmt, err := io.Copy(w, buf)
		return int(writeAmt), err
	case CompressionRle8:
		if layer == nil {
			return 0, ErrFormat("RLE compression requires layer information")
		}
		if len(layer.Fields) == 0 {
			return 0, ErrFormat("RLE compression requires layer fields to be defined")
		}
		buf := new(bytes.Buffer)
		// two modes: separated vs condensed
		if layer.Separated {
			field := layer.Fields[tileIndex/layer.Dimensions.Tiles()]
			fieldBytes := field.Size()
			for i := 0; i < len(chunk); i += fieldBytes {
				j := i + fieldBytes
				if j > len(chunk) {
					break
				}
				sample := chunk[i:j]

				// count repeats
				repeatCount := byte(1)
				for k := j; k < len(chunk); k += fieldBytes {
					l := k + fieldBytes
					if l > len(chunk) {
						break
					}
					nextSample := chunk[k:l]
					if !bytes.Equal(sample, nextSample) || repeatCount == 255 {
						break
					}
					repeatCount++
				}

				// write count and sample
				err := buf.WriteByte(repeatCount)
				if err != nil {
					return 0, err
				}
				_, err = buf.Write(sample)
				if err != nil {
					return 0, err
				}

				// advance i to skip repeats
				i += (int(repeatCount) - 1) * fieldBytes
			}
		} else {
			fieldsBytes := layer.Fields.Size()
			for i := 0; i < len(chunk); i += fieldsBytes {
				j := i + fieldsBytes
				if j > len(chunk) {
					break
				}
				sample := chunk[i:j]

				// count repeats
				repeatCount := byte(1)
				for k := j; k < len(chunk); k += fieldsBytes {
					l := k + fieldsBytes
					if l > len(chunk) {
						break
					}
					nextSample := chunk[k:l]
					if !bytes.Equal(sample, nextSample) || repeatCount == 255 {
						break
					}
					repeatCount++
				}

				// write count and sample
				err := buf.WriteByte(repeatCount)
				if err != nil {
					return 0, err
				}
				_, err = buf.Write(sample)
				if err != nil {
					return 0, err
				}

				// advance i to skip repeats
				i += (int(repeatCount) - 1) * fieldsBytes
			}
		}
		// write buffer to writer
		amtWrt, err := io.Copy(w, buf)
		return int(amtWrt), err
	default:
		return 0, ErrUnsupported("unknown compression")
	}
}

// Reads a compressed chunk of data into the given slice which must be the size of the desired
// uncompressed data. Returns the number of bytes read or and error if the read failed.
func (c Compression) readChunk(r io.Reader, layer *Layer, tileIndex int, chunk []byte) (int, error) {
	switch c {
	case CompressionNone:
		return r.Read(chunk)
	case CompressionFlate:
		bufRd := bytes.NewBuffer(chunk[:0])
		flateRdr := flate.NewReader(r)
		defer flateRdr.Close()
		amtRd, err := io.Copy(bufRd, flateRdr)
		copy(chunk, bufRd.Bytes())
		return int(amtRd), err
	case CompressionLzwLsb:
		bufRd := bytes.NewBuffer(chunk[:0])
		lzwRdr := lzw.NewReader(r, lzw.LSB, 8)
		defer lzwRdr.Close()
		amtRd, err := io.Copy(bufRd, lzwRdr)
		copy(chunk, bufRd.Bytes())
		return int(amtRd), err
	case CompressionLzwMsb:
		bufRd := bytes.NewBuffer(chunk[:0])
		lzwRdr := lzw.NewReader(r, lzw.MSB, 8)
		defer lzwRdr.Close()
		amtRd, err := io.Copy(bufRd, lzwRdr)
		copy(chunk, bufRd.Bytes())
		return int(amtRd), err
	case CompressionRle8:
		if layer == nil {
			return 0, ErrFormat("RLE compression requires layer information")
		}
		if len(layer.Fields) == 0 {
			return 0, ErrFormat("RLE compression requires layer fields to be defined")
		}
		chunkOffset := 0
		if layer.Separated {
			field := layer.Fields[tileIndex/layer.Dimensions.Tiles()]
			fieldBytes := field.Size()
			for chunkOffset < len(chunk) {
				// read repeat count
				countByte := make([]byte, 1)
				_, err := r.Read(countByte)
				if err != nil {
					return chunkOffset, err
				}
				repeatCount := int(countByte[0])
				// read sample
				sample := make([]byte, fieldBytes)
				_, err = r.Read(sample)
				if err != nil {
					return chunkOffset, err
				}
				// write sample repeatCount times
				for range repeatCount {
					copy(chunk[chunkOffset:chunkOffset+fieldBytes], sample)
					chunkOffset += fieldBytes
				}
			}
		} else {
			fieldsBytes := layer.Fields.Size()
			for chunkOffset < len(chunk) {
				// read repeat count
				countByte := make([]byte, 1)
				_, err := r.Read(countByte)
				if err != nil {
					return chunkOffset, err
				}
				repeatCount := int(countByte[0])
				// read sample
				sample := make([]byte, fieldsBytes)
				_, err = r.Read(sample)
				if err != nil {
					return chunkOffset, err
				}
				// write sample repeatCount times
				for range repeatCount {
					copy(chunk[chunkOffset:], sample)
					chunkOffset += fieldsBytes
				}
			}
		}
		return chunkOffset, nil
	default:
		return 0, ErrUnsupported("unknown compression")
	}
}
