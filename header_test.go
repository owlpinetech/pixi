package pixi

import (
	"bytes"
	"encoding/binary"
	"math/rand/v2"
	"testing"

	"github.com/owlpinetech/pixi/internal/buffer"
)

func TestWriteReadHeader(t *testing.T) {
	baseCases := []PixiHeader{
		{Version: Version, OffsetSize: 8, ByteOrder: binary.LittleEndian},
		{Version: Version, OffsetSize: 4, ByteOrder: binary.LittleEndian},
		{Version: Version, OffsetSize: 8, ByteOrder: binary.BigEndian},
		{Version: Version, OffsetSize: 4, ByteOrder: binary.BigEndian},
	}

	for range 10 {
		for _, header := range baseCases {
			if header.OffsetSize == 4 {
				header.FirstLayerOffset = int64(rand.Int32())
				header.FirstTagsOffset = int64(rand.Int32())
			} else {
				header.FirstLayerOffset = rand.Int64()
				header.FirstTagsOffset = rand.Int64()
			}

			buf := buffer.NewBuffer(10)
			err := (&header).WriteHeader(buf)
			if err != nil {
				t.Fatal(err)
			}

			rdBuf := bytes.NewReader(buf.Bytes())
			rdHeader := &PixiHeader{}
			err = rdHeader.ReadHeader(rdBuf)
			if err != nil {
				t.Fatal(err)
			}

			if header != *rdHeader {
				t.Errorf("read header %v was different than written header %v", *rdHeader, header)
			}

			// now change the offsets and read again
			if header.OffsetSize == 4 {
				header.OverwriteOffsets(buf, int64(rand.Int32()), int64(rand.Int32()))
			} else {
				header.OverwriteOffsets(buf, rand.Int64(), rand.Int64())
			}

			rdBuf = bytes.NewReader(buf.Bytes())
			rdHeader = &PixiHeader{}
			err = rdHeader.ReadHeader(rdBuf)
			if err != nil {
				t.Fatal(err)
			}

			if header != *rdHeader {
				t.Errorf("read header %v was different than written header with new offsets %v", *rdHeader, header)
			}
		}
	}
}
