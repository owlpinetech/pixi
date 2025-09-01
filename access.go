package pixi

import "io"

type DirectAccessLayerReader interface {
	SampleAt(backing io.ReadSeeker, coord SampleCoordinate) ([]any, error)
	FieldAt(backing io.ReadSeeker, coord SampleCoordinate, fieldIndex int) (any, error)
}

type DirectAccessLayerWriter interface {
	SetSampleAt(backing io.WriteSeeker, coord SampleCoordinate, values []any) error
	SetFieldAt(backing io.WriteSeeker, coord SampleCoordinate, fieldIndex int, value any) error
}

type DirectAccessLayerReadWriter interface {
	DirectAccessLayerReader
	DirectAccessLayerWriter
}

type IterativeLayerReader interface {
	NextField(backing io.ReadSeeker) (SampleCoordinate, any, error)
	NextSample(backing io.ReadSeeker) (SampleCoordinate, []any, error)
	NamedNextSample(backing io.ReadSeeker) (SampleCoordinate, map[string]any, error)
}

type IterativeLayerWriter interface {
	SetNextField(backing io.WriteSeeker, value any) error
	SetNextSample(backing io.WriteSeeker, values []any) error
	SetNamedNextSample(backing io.WriteSeeker, values map[string]any) error
}

type IterativeLayerReadWriter interface {
	IterativeLayerReader
	IterativeLayerWriter
}
