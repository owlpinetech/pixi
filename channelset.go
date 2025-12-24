package pixi

// An ordered set of named channels present in each sample of a layer in a Pixi file.
type ChannelSet []Channel

// The size in bytes of each sample in the data set. Each channel has a fixed size, and a sample
// is made up of one element of each channel, so the sample size is the sum of all channel sizes.
func (set ChannelSet) Size() int {
	sampleSize := 0
	for _, c := range set {
		sampleSize += c.Size()
	}
	return sampleSize
}

// The index of the (first) channel with the given name in the set, or -1 if not found.
func (set ChannelSet) Index(channelName string) int {
	for i, channel := range set {
		if channel.Name == channelName {
			return i
		}
	}
	return -1
}

// The byte offset of the channel within a given sample. This is the sum of the sizes of all preceding channels.
func (set ChannelSet) Offset(channelIndex int) int {
	offset := 0
	for i := range channelIndex {
		offset += set[i].Size()
	}
	return offset
}

// The byte offset of the channel with the given name within a sample. This is the sum of the sizes of all preceding channels.
// Panics if the channel is not found in the set.
func (set ChannelSet) NamedOffset(channelName string) int {
	offset := 0
	for _, channel := range set {
		if channel.Name == channelName {
			return offset
		}
		offset += channel.Size()
	}
	panic("pixi: channel not found in channel set")
}
