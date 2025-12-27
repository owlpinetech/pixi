package pixi

// A sample is a list of channel values, in channel-index order, for at a single index / coordinate in a layer.
type Sample []any

// Creates a Sample from a map of named channel values, according to the order of channels in the given layer.
func FromNamedSample(channelset ChannelSet, named map[string]any) Sample {
	sample := make(Sample, len(channelset))
	for i, channel := range channelset {
		sample[i] = named[channel.Name]
	}
	return sample
}

// Creates a map of named channel values from the Sample, according to the order of channels in the given layer.
func (s Sample) Named(channelSet ChannelSet) map[string]any {
	named := make(map[string]any)
	for i, channel := range channelSet {
		named[channel.Name] = s[i]
	}
	return named
}
