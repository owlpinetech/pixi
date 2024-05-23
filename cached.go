package pixi

import "io"

type CacheTile struct {
	Data []byte
}

type CacheDataset struct {
	DataSet
	Tiles  map[int]CacheTile
	Reader io.ReadSeeker
}

func ReadCached(r io.ReadSeeker, ds DataSet) (CacheDataset, error) {
	cached := CacheDataset{DataSet: ds, Tiles: make(map[int]CacheTile), Reader: r}
	return cached, nil
}
