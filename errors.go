package pixi

import "fmt"

type ErrFormat string

func (e ErrFormat) Error() string {
	return "pixi: format error - " + string(e)
}

type ErrUnsupported string

func (e ErrUnsupported) Error() string {
	return "pixi: unsupported action - " + string(e)
}

type ErrDataIntegrity struct {
	TileIndex int
	LayerName string
}

func (e ErrDataIntegrity) Error() string {
	return fmt.Sprintf("pixi: data integrity compromised - tile %d, layer '%s'", e.TileIndex, e.LayerName)
}

type ErrFieldNotFound struct {
	FieldName string
}

func (e ErrFieldNotFound) Error() string {
	return fmt.Sprintf("pixi: field not found - '%s'", e.FieldName)
}

type ErrTileNotFound struct {
	TileIndex int
}

func (e ErrTileNotFound) Error() string {
	return fmt.Sprintf("pixi: tile not found - index %d", e.TileIndex)
}

type ErrSampleCoordinateOutOfBounds struct {
	Coordinate SampleCoordinate
	Dimensions DimensionSet
}

func (e ErrSampleCoordinateOutOfBounds) Error() string {
	return fmt.Sprintf("pixi: sample coordinate out of bounds - coordinate %v, dimensions %v", e.Coordinate, e.Dimensions)
}
