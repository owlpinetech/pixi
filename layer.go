package pixi

import "io"

type Layer struct {
	Name string // Friendly name of the layer
	// Indicates whether the fields of the dataset are stored separated or contiguously. If true,
	// values for each field are stored next to each other. If false, the default, values for each
	// index are stored next to each other, with values for different fields stored next to each
	// other at the same index.
	Separated   bool
	Compression Compression // The type of compression used on this dataset (e.g., Flate, lz4).
	// An array of Dimension structs representing the dimensions and tiling of this dataset.
	// No dimensions equals an empty dataset.
	Dimensions []Dimension
	Fields     []Field // An array of Field structs representing the fields in this dataset.
}

// Computes the number of non-separated tiles in the data set. This number is the same regardless
// of how the tiles are laid out on disk; use the DiskTiles() method to determine the number of
// tiles actually stored on disk. Note that DiskTiles() >= Tiles() by definition.
func (d *Layer) Tiles() int {
	tiles := 1
	for _, t := range d.Dimensions {
		tiles *= t.Tiles()
	}
	return tiles
}

// The number of samples per tile in the data set. Each tile has the same number of samples,
// regardless of if the data is stored separated or continguous.
func (d *Layer) TileSamples() int {
	if len(d.Dimensions) <= 0 {
		return 0
	}
	samples := 1
	for _, d := range d.Dimensions {
		samples *= d.TileSize
	}
	return samples
}

// The total number of samples in the data set. If the tile size of any dimension is not
// a multiple of the dimension size, the 'padding' samples are not included in the count.
func (d *Layer) Samples() int {
	if len(d.Dimensions) <= 0 {
		return 0
	}
	samples := 1
	for _, dim := range d.Dimensions {
		samples *= dim.Size
	}
	return samples
}

// The size of the requested disk tile in bytes. For contiguous files, the size of each tile is always
// the same. However, for separated data sets, each field is tiled (so the number of on-disk
// tiles is actually fieldCount * Tiles()). Hence, the tile size changes depending on which
// field is being accessed.
func (d *Layer) DiskTileSize(tileIndex int) int {
	if d.Tiles() == 0 {
		return 0
	}
	if d.Separated {
		field := tileIndex / d.Tiles()
		return d.TileSamples() * d.Fields[field].Size()
	} else {
		return d.TileSamples() * d.SampleSize()
	}
}

// The number of discrete data tiles actually stored in the backing file. This number differs based
// on whether fields are stored 'contiguous' or 'separated'; in the former case, DiskTiles() == Tiles(),
// in the latter case, DiskTiles() == Tiles() * number of fields.
func (d *Layer) DiskTiles() int {
	tiles := d.Tiles()
	if d.Separated {
		tiles *= len(d.Fields)
	}
	return tiles
}

// The size in bytes of each sample in the data set. Each field has a fixed size, and a sample
// is made up of one element of each field, so the sample size is the sum of all field sizes.
func (d *Layer) SampleSize() int {
	sampleSize := 0
	for _, f := range d.Fields {
		sampleSize += f.Size()
	}
	return sampleSize
}

// Pixi files are composed of one or more layers. Generally, layers are used to represent the same data set
// at different 'zoom levels'. For example, a large digital elevation model data set might have a layer
// that shows a zoomed-out view of the terrain at a much smaller footprint, useful for thumbnails and previews.
// Layers are also useful if data sets of different resolutions should be stored together in the same file.
type DiskLayer struct {
	Layer
	TileBytes      []int64 // An array of byte counts representing (compressed) size of each tile in bytes for this dataset.
	TileOffsets    []int64 // An array of byte offsets representing the position in the file of each tile in the dataset.
	NextLayerStart int64   // The start of the next layer in the file, in units of bytes. 0 if this is the last layer in the file.
}

func (d *DiskLayer) DiskHeaderSize() int64 {
	headerSize := int64(4)                       // config (separated only currently) is 4 bytes
	headerSize += 4 * 3                          // 4 bytes for compression, dim count, field count
	headerSize += 4 + int64(len([]byte(d.Name))) // 4 bytes for name length, then name
	headerSize += int64(len(d.Dimensions)) * 8   // 8 bytes for each dimension size
	headerSize += int64(len(d.Dimensions)) * 8   // 8 bytes for each dimension tile size
	headerSize += int64(len(d.Fields)) * 4       // 4 bytes for each field type
	headerSize += int64(len(d.Fields)) * 2       // 2 bytes for each field name length
	for _, f := range d.Fields {
		headerSize += int64(len([]byte(f.Name))) // each field name length in bytes
	}
	headerSize += int64(d.DiskTiles()) * 8 // 8 bytes for each real disk tile size in bytes
	headerSize += int64(d.DiskTiles()) * 8 // 8 bytes for each tile offset
	headerSize += 8                        // 8 bytes for the next layer start offset
	return headerSize
}

// The on-disk size in bytes of the (potentially compressed) data set. Does not include the dataset
// header size.
func (d *DiskLayer) DataSize() int64 {
	size := int64(0)
	for _, b := range d.TileBytes {
		size += b
	}
	return size
}

// Compacts the tiles in the layer so that as few bytes on disk as possible are wasted, and moves
// the whole layer to the specified offset in the file. If there is a write or read error during
// compaction, the process stops immediately and returns the error. Otherwise, the new end offset
// of the layer is returned as the result of compaction and moving.
func (d *DiskLayer) MoveAndCompact(backing io.ReadWriteSeeker, newOffset int64) (int64, error) {

}

func (l *DiskLayer) FillBlank(backing io.ReadWriteSeeker) error {

}

func (l *DiskLayer) WriteInOrder(backing io.ReadWriteSeeker, iter func(index []uint) []any) error {

}
