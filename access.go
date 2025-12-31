package pixi

type Accessor interface {
	Layer() Layer
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
	layer := accessor.Layer()
	tileSelector := coord.ToTileSelector(layer.Dimensions)
	sample := make([]any, len(layer.Channels))

	if layer.Separated {
		for channelIndex, channel := range layer.Channels {
			channelTile := tileSelector.Tile + layer.Dimensions.Tiles()*channelIndex

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
		channelOffset := tileSelector.InTile * layer.Channels.Size()

		tileData, err := accessor.Tile(tileSelector.Tile)
		if err != nil {
			return nil, err
		}
		for i, channel := range layer.Channels {
			sample[i] = channel.Value(tileData[channelOffset:], accessor.Header().ByteOrder)
			channelOffset += channel.Size()
		}
	}

	return sample, nil
}

func ChannelAt(accessor TileAccessLayer, coord SampleCoordinate, channelIndex int) (any, error) {
	layer := accessor.Layer()
	tileSelector := coord.ToTileSelector(layer.Dimensions)
	channel := layer.Channels[channelIndex]

	if layer.Separated {
		channelTile := tileSelector.Tile + layer.Dimensions.Tiles()*channelIndex

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
		channelOffset := tileSelector.InTile * layer.Channels.Size()
		for _, channel := range layer.Channels[:channelIndex] {
			channelOffset += channel.Size()
		}
		return channel.Value(tileData[channelOffset:], accessor.Header().ByteOrder), nil
	}
}

func SetSampleAt(modifier TileModifierLayer, coord SampleCoordinate, values Sample) error {
	layer := modifier.Layer()
	// Update Min/Max for all channels
	for channelIndex, value := range values {
		layer.Channels[channelIndex] = layer.Channels[channelIndex].WithMinMax(value)
	}

	tileSelector := coord.ToTileSelector(layer.Dimensions)

	if layer.Separated {
		for channelIndex, channel := range layer.Channels {
			channelTile := tileSelector.Tile + layer.Dimensions.Tiles()*channelIndex

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
		channelOffset := tileSelector.InTile * layer.Channels.Size()

		tileData, err := modifier.Tile(tileSelector.Tile)
		if err != nil {
			return err
		}
		for i, channel := range layer.Channels {
			channel.PutValue(values[i], modifier.Header().ByteOrder, tileData[channelOffset:])
			channelOffset += channel.Size()
		}
		modifier.SetDirty(tileSelector.Tile)
	}

	return nil
}

func SetChannelAt(modifier TileModifierLayer, coord SampleCoordinate, channelIndex int, value any) error {
	layer := modifier.Layer()
	// Update Min/Max for the channel
	layer.Channels[channelIndex] = layer.Channels[channelIndex].WithMinMax(value)

	tileSelector := coord.ToTileSelector(layer.Dimensions)
	channel := layer.Channels[channelIndex]

	if layer.Separated {
		channelTile := tileSelector.Tile + layer.Dimensions.Tiles()*channelIndex

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
		channelOffset := tileSelector.InTile * layer.Channels.Size()
		for _, channel := range layer.Channels[:channelIndex] {
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
