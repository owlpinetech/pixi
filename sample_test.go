package gopixi

import "testing"

func TestSampleFromNamedSample(t *testing.T) {
	channels := ChannelSet{
		{Name: "channel1", Type: ChannelInt32},
		{Name: "channel2", Type: ChannelInt64},
		{Name: "channel3", Type: ChannelFloat32},
	}

	named := map[string]any{
		"channel1": int32(42),
		"channel2": int64(10000000000),
		"channel3": float32(3.14),
		"channel4": "extra channel",
	}

	sample := FromNamedSample(channels, named)
	if sample[0] != int32(42) {
		t.Errorf("Expected channel1 to be 42, got %v", sample[0])
	}
	if sample[1] != int64(10000000000) {
		t.Errorf("Expected channel2 to be 10000000000, got %v", sample[1])
	}
	if sample[2] != float32(3.14) {
		t.Errorf("Expected channel3 to be 3.14, got %v", sample[2])
	}
}

func TestSampleNamed(t *testing.T) {
	channels := ChannelSet{
		{Name: "channel1", Type: ChannelInt32},
		{Name: "channel2", Type: ChannelInt64},
		{Name: "channel3", Type: ChannelFloat32},
	}

	sample := Sample{int32(42), int64(10000000000), float32(3.14)}

	named := sample.Named(channels)
	if named["channel1"] != int32(42) {
		t.Errorf("Expected channel1 to be 42, got %v", named["channel1"])
	}
	if named["channel2"] != int64(10000000000) {
		t.Errorf("Expected channel2 to be 10000000000, got %v", named["channel2"])
	}
	if named["channel3"] != float32(3.14) {
		t.Errorf("Expected channel3 to be 3.14, got %v", named["channel3"])
	}
}
