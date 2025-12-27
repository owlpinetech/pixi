package pixi

import "testing"

func TestChannelSetSize(t *testing.T) {
	tests := []struct {
		name     string
		channels ChannelSet
		wantSize int
	}{
		{
			name:     "No channels",
			channels: ChannelSet{},
			wantSize: 0,
		},
		{
			name:     "One channel with size 1",
			channels: ChannelSet{{Name: "", Type: ChannelInt8}},
			wantSize: 1,
		},
		{
			name:     "One channel with size 2",
			channels: ChannelSet{{Name: "", Type: ChannelInt16}},
			wantSize: 2,
		},
		{
			name:     "Multiple channels with different sizes",
			channels: ChannelSet{{Name: "", Type: ChannelInt8}, {Name: "", Type: ChannelFloat32}},
			wantSize: 5, // size of int8 + size of float32
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotSize := test.channels.Size()
			if gotSize != test.wantSize {
				t.Errorf("channels.Size() = %d, want %d", gotSize, test.wantSize)
			}
		})
	}
}

func TestChannelSetIndex(t *testing.T) {
	tests := []struct {
		name        string
		channels    ChannelSet
		channelName string
		wantIndex   int
	}{
		{
			name:        "Channel exists at index 0",
			channels:    ChannelSet{{Name: "channel1", Type: ChannelInt8}, {Name: "channel2", Type: ChannelInt16}},
			channelName: "channel1",
			wantIndex:   0,
		},
		{
			name:        "Channel exists at index 1",
			channels:    ChannelSet{{Name: "channel1", Type: ChannelInt8}, {Name: "channel2", Type: ChannelInt16}},
			channelName: "channel2",
			wantIndex:   1,
		},
		{
			name:        "Channel does not exist",
			channels:    ChannelSet{{Name: "channel1", Type: ChannelInt8}, {Name: "channel2", Type: ChannelInt16}},
			channelName: "channel3",
			wantIndex:   -1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotIndex := test.channels.Index(test.channelName)
			if gotIndex != test.wantIndex {
				t.Errorf("channels.Index(%q) = %d, want %d", test.channelName, gotIndex, test.wantIndex)
			}
		})
	}
}

func TestChannelSetOffset(t *testing.T) {
	tests := []struct {
		name         string
		channels     ChannelSet
		channelIndex int
		wantOffset   int
	}{
		{
			name:         "Offset of first channel",
			channels:     ChannelSet{{Name: "channel1", Type: ChannelInt8}, {Name: "channel2", Type: ChannelInt16}},
			channelIndex: 0,
			wantOffset:   0,
		},
		{
			name:         "Offset of second channel",
			channels:     ChannelSet{{Name: "channel1", Type: ChannelInt8}, {Name: "channel2", Type: ChannelInt16}},
			channelIndex: 1,
			wantOffset:   1, // size of int8
		},
		{
			name:         "Offset of third channel",
			channels:     ChannelSet{{Name: "channel1", Type: ChannelInt8}, {Name: "channel2", Type: ChannelInt16}, {Name: "channel3", Type: ChannelFloat32}},
			channelIndex: 2,
			wantOffset:   3, // size of int8 + size of int16
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotOffset := test.channels.Offset(test.channelIndex)
			if gotOffset != test.wantOffset {
				t.Errorf("channels.Offset(%d) = %d, want %d", test.channelIndex, gotOffset, test.wantOffset)
			}
		})
	}
}

func TestChannelSetNamedOffset(t *testing.T) {
	tests := []struct {
		name        string
		channels    ChannelSet
		channelName string
		wantOffset  int
		wantPanic   bool
	}{
		{
			name:        "Offset of first channel",
			channels:    ChannelSet{{Name: "channel1", Type: ChannelInt8}, {Name: "channel2", Type: ChannelInt16}},
			channelName: "channel1",
			wantOffset:  0,
			wantPanic:   false,
		},
		{
			name:        "Offset of second channel",
			channels:    ChannelSet{{Name: "channel1", Type: ChannelInt8}, {Name: "channel2", Type: ChannelInt16}},
			channelName: "channel2",
			wantOffset:  1, // size of int8
			wantPanic:   false,
		},
		{
			name:        "Offset of third channel",
			channels:    ChannelSet{{Name: "channel1", Type: ChannelInt8}, {Name: "channel2", Type: ChannelInt16}, {Name: "channel3", Type: ChannelFloat32}},
			channelName: "channel3",
			wantOffset:  3, // size of int8 + size of int16
			wantPanic:   false,
		},
		{
			name:        "Channel does not exist",
			channels:    ChannelSet{{Name: "channel1", Type: ChannelInt8}, {Name: "channel2", Type: ChannelInt16}},
			channelName: "channel3",
			wantOffset:  0,
			wantPanic:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !test.wantPanic {
						t.Errorf("did not expect to panic for test")
					}
				}
			}()

			gotOffset := test.channels.NamedOffset(test.channelName)
			if gotOffset != test.wantOffset {
				t.Errorf("channels.NamedOffset(%q) = %d, want %d", test.channelName, gotOffset, test.wantOffset)
			}
		})
	}
}
