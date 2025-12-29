package pixi

type Accessor interface {
	Layer() *Layer
}

type TileAccessLayer interface {
	Accessor
	Header() Header
	Tile(tile int) ([]byte, error)
}

type TileModifierLayer interface {
	TileAccessLayer
	SetDirty(tile int)
	Commit() error
}

func SampleAt(accessor TileAccessLayer, coord SampleCoordinate) (Sample, error) {
	tileSelector := coord.ToTileSelector(accessor.Layer().Dimensions)
	sample := make([]any, len(accessor.Layer().Channels))

	if accessor.Layer().Separated {
		for channelIndex, channel := range accessor.Layer().Channels {
			channelTile := tileSelector.Tile + accessor.Layer().Dimensions.Tiles()*channelIndex

			tileData, err := accessor.Tile(channelTile)
			if err != nil {
				return nil, err
			}

			if channel.Type == ChannelBool {
				sample[channelIndex] = UnpackBool(tileData, tileSelector.InTile)
			} else {
				channelOffset := tileSelector.InTile * channel.Size()
				sample[channelIndex] = channel.Value(tileData[channelOffset:], accessor.Header().ByteOrder)
			}
		}
	} else {
		channelOffset := tileSelector.InTile * accessor.Layer().Channels.Size()

		tileData, err := accessor.Tile(tileSelector.Tile)
		if err != nil {
			return nil, err
		}
		for i, channel := range accessor.Layer().Channels {
			sample[i] = channel.Value(tileData[channelOffset:], accessor.Header().ByteOrder)
			channelOffset += channel.Size()
		}
	}

	return sample, nil
}

func ChannelAt(accessor TileAccessLayer, coord SampleCoordinate, channelIndex int) (any, error) {
	tileSelector := coord.ToTileSelector(accessor.Layer().Dimensions)
	channel := accessor.Layer().Channels[channelIndex]

	if accessor.Layer().Separated {
		channelTile := tileSelector.Tile + accessor.Layer().Dimensions.Tiles()*channelIndex

		tileData, err := accessor.Tile(channelTile)
		if err != nil {
			return nil, err
		}

		if channel.Type == ChannelBool {
			return UnpackBool(tileData, tileSelector.InTile), nil
		} else {
			channelOffset := tileSelector.InTile * channel.Size()
			return channel.Value(tileData[channelOffset:], accessor.Header().ByteOrder), nil
		}
	} else {
		tileData, err := accessor.Tile(tileSelector.Tile)
		if err != nil {
			return nil, err
		}
		channelOffset := tileSelector.InTile * accessor.Layer().Channels.Size()
		for _, channel := range accessor.Layer().Channels[:channelIndex] {
			channelOffset += channel.Size()
		}
		return channel.Value(tileData[channelOffset:], accessor.Header().ByteOrder), nil
	}
}

func SetSampleAt(modifier TileModifierLayer, coord SampleCoordinate, values Sample) error {
	// Update Min/Max for all channels
	for channelIndex, value := range values {
		modifier.Layer().Channels[channelIndex].UpdateMinMax(value)
	}

	tileSelector := coord.ToTileSelector(modifier.Layer().Dimensions)

	if modifier.Layer().Separated {
		for channelIndex, channel := range modifier.Layer().Channels {
			channelTile := tileSelector.Tile + modifier.Layer().Dimensions.Tiles()*channelIndex

			tileData, err := modifier.Tile(channelTile)
			if err != nil {
				return err
			}
			if channel.Type == ChannelBool {
				PackBool(values[channelIndex].(bool), tileData, tileSelector.InTile)
			} else {
				channelOffset := tileSelector.InTile * channel.Size()
				channel.PutValue(values[channelIndex], modifier.Header().ByteOrder, tileData[channelOffset:])
			}
			modifier.SetDirty(channelTile)
		}
	} else {
		channelOffset := tileSelector.InTile * modifier.Layer().Channels.Size()

		tileData, err := modifier.Tile(tileSelector.Tile)
		if err != nil {
			return err
		}
		for i, channel := range modifier.Layer().Channels {
			channel.PutValue(values[i], modifier.Header().ByteOrder, tileData[channelOffset:])
			channelOffset += channel.Size()
		}
		modifier.SetDirty(tileSelector.Tile)
	}

	return nil
}

func SetChannelAt(modifier TileModifierLayer, coord SampleCoordinate, channelIndex int, value any) error {
	// Update Min/Max for the channel
	modifier.Layer().Channels[channelIndex].UpdateMinMax(value)

	tileSelector := coord.ToTileSelector(modifier.Layer().Dimensions)
	channel := modifier.Layer().Channels[channelIndex]

	if modifier.Layer().Separated {
		channelTile := tileSelector.Tile + modifier.Layer().Dimensions.Tiles()*channelIndex

		tileData, err := modifier.Tile(channelTile)
		if err != nil {
			return err
		}

		if channel.Type == ChannelBool {
			PackBool(value.(bool), tileData, tileSelector.InTile)
		} else {
			channelOffset := tileSelector.InTile * channel.Size()
			channel.PutValue(value, modifier.Header().ByteOrder, tileData[channelOffset:])
		}
		modifier.SetDirty(channelTile)
	} else {
		tileData, err := modifier.Tile(tileSelector.Tile)
		if err != nil {
			return err
		}
		channelOffset := tileSelector.InTile * modifier.Layer().Channels.Size()
		for _, channel := range modifier.Layer().Channels[:channelIndex] {
			channelOffset += channel.Size()
		}
		channel.PutValue(value, modifier.Header().ByteOrder, tileData[channelOffset:])
		modifier.SetDirty(tileSelector.Tile)
	}
	return nil
}

type IterativeLayer interface {
	Accessor
	Done()
	Next() bool
	Error() error
	Coordinate() SampleCoordinate
}

type IterativeLayerReader interface {
	IterativeLayer
	Channel(channelIndex int) any
	Sample() Sample
}

type IterativeLayerWriter interface {
	IterativeLayer
	SetChannel(channelIndex int, value any)
	SetSample(values Sample)
}

type IterativeLayerReadWriter interface {
	IterativeLayerReader
	IterativeLayerWriter
}
