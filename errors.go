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

type IntegrityError struct {
	TileIndex int
	LayerName string
}

func (e IntegrityError) Error() string {
	return fmt.Sprintf("pixi: data integrity compromised - tile %d, layer '%s'", e.TileIndex, e.LayerName)
}
