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
		amtWrt, err := CompressionFlate.writeChunk(buf, nil, 0, chunk)
		if err != nil {
			t.Fatal(err)
		}

		if amtWrt < 1 {
			t.Error("expected write amount to be more than 0")
		}

		rdr := bytes.NewReader(buf.Bytes())
		rdChunk := make([]byte, len(chunk))
		amtRcv, err := CompressionFlate.readChunk(rdr, nil, 0, rdChunk)
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
		amtWrt, err := CompressionLzwLsb.writeChunk(buf, nil, 0, chunk)
		if err != nil {
			t.Fatal(err)
		}

		if amtWrt < 1 {
			t.Error("expected write amount to be more than 0")
		}

		rdr := bytes.NewReader(buf.Bytes())
		rdChunk := make([]byte, len(chunk))
		amtRcv, err := CompressionLzwLsb.readChunk(rdr, nil, 0, rdChunk)
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
		amtWrt, err := CompressionLzwMsb.writeChunk(buf, nil, 0, chunk)
		if err != nil {
			t.Fatal(err)
		}

		if amtWrt < 1 {
			t.Error("expected write amount to be more than 0")
		}

		rdr := bytes.NewReader(buf.Bytes())
		rdChunk := make([]byte, len(chunk))
		amtRcv, err := CompressionLzwMsb.readChunk(rdr, nil, 0, rdChunk)
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

func TestRle8CompressionWriteReadCondensedLayer(t *testing.T) {
	for range 25 {
		// create between 1 and 5 channels of random sizes
		channelCount := rand.IntN(5) + 1
		channels := make(ChannelSet, channelCount)
		for i := range channelCount {
			channelSize := (rand.IntN(2) + 1) * 2
			channelType := ChannelUint8
			switch channelSize {
			case 1:
				channelType = ChannelUint8
			case 2:
				channelType = ChannelUint16
			case 4:
				channelType = ChannelUint32
			}
			channels[i] = Channel{
				Name: "channel-" + string(rune('A'+i)),
				Type: channelType,
			}
		}

		// create a chunk with runs of repeated bytes
		chunk := []byte{}
		for range 50 {
			repeatCount := rand.IntN(10) + 1
			sample := make([]byte, channels.Size())
			for i := range sample {
				sample[i] = byte(rand.IntN(256))
			}
			for range repeatCount {
				chunk = append(chunk, sample...)
			}
		}

		buf := bytes.NewBuffer([]byte{})
		layer := &Layer{Channels: channels, Separated: false}
		amtWrt, err := CompressionRle8.writeChunk(buf, layer, 0, chunk)
		if err != nil {
			t.Fatal(err)
		}

		if amtWrt < 1 {
			t.Error("expected write amount to be more than 0")
		}

		rdr := bytes.NewReader(buf.Bytes())
		rdChunk := make([]byte, len(chunk))
		amtRcv, err := CompressionRle8.readChunk(rdr, layer, 0, rdChunk)
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
