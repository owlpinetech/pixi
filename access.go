package pixi

type LayerExtension interface {
	Layer() *Layer
}

type DirectAccessLayerReader interface {
	LayerExtension
	SampleAt(coord SampleCoordinate) (Sample, error)
	FieldAt(coord SampleCoordinate, fieldIndex int) (any, error)
}

type DirectAccessLayerWriter interface {
	LayerExtension
	SetSampleAt(coord SampleCoordinate, values Sample) error
	SetFieldAt(coord SampleCoordinate, fieldIndex int, value any) error
	Flush() error
}

type DirectAccessLayerReadWriter interface {
	DirectAccessLayerReader
	DirectAccessLayerWriter
}

type IterativeLayer interface {
	LayerExtension
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
