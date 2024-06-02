package pixi

import "fmt"

type FormatError string

func (e FormatError) Error() string {
	return "pixi: format error - " + string(e)
}

type UnsupportedError string

func (e UnsupportedError) Error() string {
	return "pixi: unsupported action - " + string(e)
}

type DimensionsError struct {
	GivenDims  int
	ActualDims int
}

func (d DimensionsError) Error() string {
	return fmt.Sprintf("pixi: given dimension count %d did not match dataset dimesions count %d", d.GivenDims, d.ActualDims)
}

type RangeError struct {
	Specified int
	ValidMin  int
	ValidMax  int
}

func (e RangeError) Error() string {
	return fmt.Sprintf("pixi: range access error - %d was not in range [%d,%d]", e.Specified, e.ValidMin, e.ValidMax)
}
