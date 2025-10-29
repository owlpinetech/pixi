package pixi

import "io"

type LayerExtension interface {
	Layer() *Layer
}

type DirectAccessLayerReader interface {
	LayerExtension
	SampleAt(coord SampleCoordinate) ([]any, error)
	FieldAt(coord SampleCoordinate, fieldIndex int) (any, error)
}

type DirectAccessLayerWriter interface {
	LayerExtension
	SetSampleAt(coord SampleCoordinate, values []any) error
	SetFieldAt(coord SampleCoordinate, fieldIndex int, value any) error
	Flush() error
}

type DirectAccessLayerReadWriter interface {
	DirectAccessLayerReader
	DirectAccessLayerWriter
}

type IterativeLayerReader interface {
	LayerExtension
	NextField(backing io.ReadSeeker) (SampleCoordinate, any, error)
	NextSample(backing io.ReadSeeker) (SampleCoordinate, []any, error)
	NamedNextSample(backing io.ReadSeeker) (SampleCoordinate, map[string]any, error)
}

type IterativeLayerWriter interface {
	LayerExtension
	SetNextField(backing io.WriteSeeker, value any) error
	SetNextSample(backing io.WriteSeeker, values []any) error
	SetNamedNextSample(backing io.WriteSeeker, values map[string]any) error
}

type IterativeLayerReadWriter interface {
	IterativeLayerReader
	IterativeLayerWriter
}
