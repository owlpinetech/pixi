package pixi

type Dimension64Iterator struct {
	Value []int64
	Dims  []Dimension
}

func NewDimensionsIterator(dims []Dimension) *Dimension64Iterator {
	return &Dimension64Iterator{
		Value: make([]int64, len(dims)),
		Dims:  dims,
	}
}

func (i *Dimension64Iterator) Next() {
	for dInd := 0; dInd < len(i.Value); dInd++ {
		i.Value[dInd] += 1
		if i.Value[dInd] >= i.Dims[dInd].Size {
			i.Value[dInd] = 0
			if dInd < len(i.Value)-1 {
				i.Value[dInd] += 1
			}
		} else {
			break
		}
	}
}

func (i *Dimension64Iterator) HasNext() bool {
	for dInd := 0; dInd < len(i.Value); dInd++ {
		if i.Value[dInd] < i.Dims[dInd].Size-1 {
			return false
		}
	}
	return true
}

type DimensionU32Iterator struct {
	Value []uint
	Dims  []Dimension
}

func NewTileIndexIterator(dims []Dimension) *DimensionU32Iterator {
	return &DimensionU32Iterator{
		Value: make([]uint, len(dims)),
		Dims:  dims,
	}
}

func (i *DimensionU32Iterator) Next() {
	for dInd := 0; dInd < len(i.Value); dInd++ {
		i.Value[dInd] += 1
		if i.Value[dInd] >= uint(i.Dims[dInd].Tiles()) {
			i.Value[dInd] = 0
			if dInd < len(i.Value)-1 {
				i.Value[dInd] += 1
			}
		} else {
			break
		}
	}
}

func (i *DimensionU32Iterator) HasNext() bool {
	for dInd := 0; dInd < len(i.Value); dInd++ {
		if i.Value[dInd] < uint(i.Dims[dInd].Tiles())-1 {
			return false
		}
	}
	return true
}
