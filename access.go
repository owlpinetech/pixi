package pixi

type Accessor interface {
	Layer() *Layer
}

type TileAccessLayer interface {
	Accessor
	Header() *Header
	Tile(tile int) ([]byte, error)
}

type TileModifierLayer interface {
	TileAccessLayer
	SetDirty(tile int)
	Commit() error
}

func SampleAt(accessor TileAccessLayer, coord SampleCoordinate) (Sample, error) {
	tileSelector := coord.ToTileSelector(accessor.Layer().Dimensions)
	sample := make([]any, len(accessor.Layer().Fields))

	if accessor.Layer().Separated {
		for fieldIndex, field := range accessor.Layer().Fields {
			fieldTile := tileSelector.Tile + accessor.Layer().Dimensions.Tiles()*fieldIndex

			tileData, err := accessor.Tile(fieldTile)
			if err != nil {
				return nil, err
			}

			if field.Type == FieldBool {
				sample[fieldIndex] = UnpackBool(tileData, tileSelector.InTile)
			} else {
				fieldOffset := tileSelector.InTile * field.Size()
				sample[fieldIndex] = field.Value(tileData[fieldOffset:], accessor.Header().ByteOrder)
			}
		}
	} else {
		fieldOffset := tileSelector.InTile * accessor.Layer().Fields.Size()

		tileData, err := accessor.Tile(tileSelector.Tile)
		if err != nil {
			return nil, err
		}
		for i, field := range accessor.Layer().Fields {
			sample[i] = field.Value(tileData[fieldOffset:], accessor.Header().ByteOrder)
			fieldOffset += field.Size()
		}
	}

	return sample, nil
}

func FieldAt(accessor TileAccessLayer, coord SampleCoordinate, fieldIndex int) (any, error) {
	tileSelector := coord.ToTileSelector(accessor.Layer().Dimensions)
	field := accessor.Layer().Fields[fieldIndex]

	if accessor.Layer().Separated {
		fieldTile := tileSelector.Tile + accessor.Layer().Dimensions.Tiles()*fieldIndex

		tileData, err := accessor.Tile(fieldTile)
		if err != nil {
			return nil, err
		}

		if field.Type == FieldBool {
			return UnpackBool(tileData, tileSelector.InTile), nil
		} else {
			fieldOffset := tileSelector.InTile * field.Size()
			return field.Value(tileData[fieldOffset:], accessor.Header().ByteOrder), nil
		}
	} else {
		tileData, err := accessor.Tile(tileSelector.Tile)
		if err != nil {
			return nil, err
		}
		fieldOffset := tileSelector.InTile * accessor.Layer().Fields.Size()
		for _, field := range accessor.Layer().Fields[:fieldIndex] {
			fieldOffset += field.Size()
		}
		return field.Value(tileData[fieldOffset:], accessor.Header().ByteOrder), nil
	}
}

func SetSampleAt(modifier TileModifierLayer, coord SampleCoordinate, values Sample) error {
	if len(values) != len(modifier.Layer().Fields) {
		panic("pixi: values length does not match field count")
	}

	// Update Min/Max for all fields
	for fieldIndex, value := range values {
		modifier.Layer().Fields[fieldIndex].UpdateMinMax(value)
	}

	tileSelector := coord.ToTileSelector(modifier.Layer().Dimensions)

	if modifier.Layer().Separated {
		for fieldIndex, field := range modifier.Layer().Fields {
			fieldTile := tileSelector.Tile + modifier.Layer().Dimensions.Tiles()*fieldIndex

			tileData, err := modifier.Tile(fieldTile)
			if err != nil {
				return err
			}
			if field.Type == FieldBool {
				PackBool(values[fieldIndex].(bool), tileData, tileSelector.InTile)
			} else {
				fieldOffset := tileSelector.InTile * field.Size()
				field.PutValue(values[fieldIndex], modifier.Header().ByteOrder, tileData[fieldOffset:])
			}
			modifier.SetDirty(fieldTile)
		}
	} else {
		fieldOffset := tileSelector.InTile * modifier.Layer().Fields.Size()

		tileData, err := modifier.Tile(tileSelector.Tile)
		if err != nil {
			return err
		}
		for i, field := range modifier.Layer().Fields {
			field.PutValue(values[i], modifier.Header().ByteOrder, tileData[fieldOffset:])
			fieldOffset += field.Size()
		}
		modifier.SetDirty(tileSelector.Tile)
	}

	return nil
}

func SetFieldAt(modifier TileModifierLayer, coord SampleCoordinate, fieldIndex int, value any) error {
	if fieldIndex < 0 || fieldIndex >= len(modifier.Layer().Fields) {
		panic("pixi: field index out of range")
	}

	// Update Min/Max for the field
	modifier.Layer().Fields[fieldIndex].UpdateMinMax(value)

	tileSelector := coord.ToTileSelector(modifier.Layer().Dimensions)
	field := modifier.Layer().Fields[fieldIndex]

	if modifier.Layer().Separated {
		fieldTile := tileSelector.Tile + modifier.Layer().Dimensions.Tiles()*fieldIndex

		tileData, err := modifier.Tile(fieldTile)
		if err != nil {
			return err
		}

		if field.Type == FieldBool {
			PackBool(value.(bool), tileData, tileSelector.InTile)
		} else {
			fieldOffset := tileSelector.InTile * field.Size()
			field.PutValue(value, modifier.Header().ByteOrder, tileData[fieldOffset:])
		}
		modifier.SetDirty(fieldTile)
	} else {
		tileData, err := modifier.Tile(tileSelector.Tile)
		if err != nil {
			return err
		}
		fieldOffset := tileSelector.InTile * modifier.Layer().Fields.Size()
		for _, field := range modifier.Layer().Fields[:fieldIndex] {
			fieldOffset += field.Size()
		}
		field.PutValue(value, modifier.Header().ByteOrder, tileData[fieldOffset:])
		modifier.SetDirty(tileSelector.Tile)
	}
	return nil
}

type DirectAccessLayerWriter interface {
	Accessor
	SetSampleAt(coord SampleCoordinate, values Sample) error
	SetFieldAt(coord SampleCoordinate, fieldIndex int, value any) error
	Flush() error
}

type DirectAccessLayerReadWriter interface {
	TileAccessLayer
	DirectAccessLayerWriter
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
	Field(fieldIndex int) any
	Sample() Sample
}

type IterativeLayerWriter interface {
	IterativeLayer
	SetField(fieldIndex int, value any)
	SetSample(values Sample)
}

type IterativeLayerReadWriter interface {
	IterativeLayerReader
	IterativeLayerWriter
}
