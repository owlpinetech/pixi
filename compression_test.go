package pixi

import (
	"bytes"
	"math/rand/v2"
	"slices"
	"testing"
)

func TestFlateCompressionWriteRead(t *testing.T) {
	for range 25 {
		chunk := make([]byte, rand.IntN(499)+1)
		for i := range len(chunk) {
			chunk[i] = byte(rand.IntN(256))
		}

		buf := bytes.NewBuffer([]byte{})
		amtWrt, err := CompressionFlate.WriteChunk(buf, chunk)
		if err != nil {
			t.Fatal(err)
		}

		if amtWrt < 1 {
			t.Error("expected write amount to be more than 0")
		}

		rdr := bytes.NewReader(buf.Bytes())
		rdChunk := make([]byte, len(chunk))
		amtRcv, err := CompressionFlate.ReadChunk(rdr, rdChunk)
		if err != nil {
			t.Fatal(err)
		}
		if amtRcv != len(chunk) {
			t.Errorf("expected to read %d bytes but read %d", len(chunk), amtRcv)
		}

		if !slices.Equal(chunk, rdChunk) {
			t.Errorf("expected chunks to be equal, got %v and %v", chunk, rdChunk)
		}
	}
}

func TestLzwLsbCompressionWriteRead(t *testing.T) {
	for range 25 {
		chunk := make([]byte, rand.IntN(499)+256)
		for i := range len(chunk) {
			chunk[i] = byte(rand.IntN(256))
		}

		buf := bytes.NewBuffer([]byte{})
		amtWrt, err := CompressionLzwLsb.WriteChunk(buf, chunk)
		if err != nil {
			t.Fatal(err)
		}

		if amtWrt < 1 {
			t.Error("expected write amount to be more than 0")
		}

		rdr := bytes.NewReader(buf.Bytes())
		rdChunk := make([]byte, len(chunk))
		amtRcv, err := CompressionLzwLsb.ReadChunk(rdr, rdChunk)
		if err != nil {
			t.Fatal(err)
		}
		if amtRcv != len(chunk) {
			t.Errorf("expected to read %d bytes but read %d", len(chunk), amtRcv)
		}

		if !slices.Equal(chunk, rdChunk) {
			t.Errorf("expected chunks to be equal, got %v and %v", chunk, rdChunk)
		}
	}
}

func TestLzwMsbCompressionWriteRead(t *testing.T) {
	for range 25 {
		chunk := make([]byte, rand.IntN(499)+256)
		for i := range len(chunk) {
			chunk[i] = byte(rand.IntN(256))
		}

		buf := bytes.NewBuffer([]byte{})
		amtWrt, err := CompressionLzwMsb.WriteChunk(buf, chunk)
		if err != nil {
			t.Fatal(err)
		}

		if amtWrt < 1 {
			t.Error("expected write amount to be more than 0")
		}

		rdr := bytes.NewReader(buf.Bytes())
		rdChunk := make([]byte, len(chunk))
		amtRcv, err := CompressionLzwMsb.ReadChunk(rdr, rdChunk)
		if err != nil {
			t.Fatal(err)
		}
		if amtRcv != len(chunk) {
			t.Errorf("expected to read %d bytes but read %d", len(chunk), amtRcv)
		}

		if !slices.Equal(chunk, rdChunk) {
			t.Errorf("expected chunks to be equal, got %v and %v", chunk, rdChunk)
		}
	}
}
