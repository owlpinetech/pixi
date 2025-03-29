package pixi

import (
	"hash/crc32"
	"io"
)

// Pixi files are composed of one or more layers. Generally, layers are used to represent the same data set
// at different 'zoom levels'. For example, a large digital elevation model data set might have a layer
// that shows a zoomed-out view of the terrain at a much smaller footprint, useful for thumbnails and previews.
// Layers are also useful if data sets of different resolutions should be stored together in the same file.
type Layer struct {
	Name string // Friendly name of the layer
	// Indicates whether the fields of the dataset are stored separated or contiguously. If true,
	// values for each field are stored next to each other. If false, the default, values for each
	// index are stored next to each other, with values for different fields stored next to each
	// other at the same index.
	Separated   bool
	Compression Compression // The type of compression used on this dataset (e.g., Flate, lz4).
	// A slice of Dimension structs representing the dimensions and tiling of this dataset.
	// No dimensions equals an empty dataset. Dimensions are stored and iterated such that the
	// samples for the first dimension are the closest together in memory, with progressively
	// higher dimensions samples becoming further apart.
	Dimensions     DimensionSet
	Fields         FieldSet // An array of Field structs representing the fields in this dataset.
	TileBytes      []int64  // An array of byte counts representing (compressed) size of each tile in bytes for this dataset.
	TileOffsets    []int64  // An array of byte offsets representing the position in the file of each tile in the dataset.
	NextLayerStart int64    // The byte-index offset of the next layer in the file, from the start of the file. 0 if this is the last layer in the file.
}

// Helper constructor to ensure that certain invariants in a layer are maintained when it is created.
func NewLayer(name string, separated bool, compression Compression, dimensions DimensionSet, fields FieldSet) *Layer {
	l := &Layer{
		Name:        name,
		Separated:   separated,
		Compression: compression,
		Dimensions:  dimensions,
		Fields:      fields,
	}

	l.TileBytes = make([]int64, l.DiskTiles())
	l.TileOffsets = make([]int64, l.DiskTiles())
	return l
}

// The size of the requested disk tile in bytes. For contiguous files, the size of each tile is always
// the same. However, for separated data sets, each field is tiled (so the number of on-disk
// tiles is actually fieldCount * Tiles()). Hence, the tile size changes depending on which
// field is being accessed.
func (d *Layer) DiskTileSize(tileIndex int) int {
	if d.Dimensions.Tiles() == 0 {
		return 0
	}
	if d.Separated {
		field := tileIndex / d.Dimensions.Tiles()
		return d.Dimensions.TileSamples() * d.Fields[field].Size()
	} else {
		return d.Dimensions.TileSamples() * d.Fields.Size()
	}
}

// The number of discrete data tiles actually stored in the backing file. This number differs based
// on whether fields are stored 'contiguous' or 'separated'; in the former case, DiskTiles() == Tiles(),
// in the latter case, DiskTiles() == Tiles() * number of fields.
func (d *Layer) DiskTiles() int {
	tiles := d.Dimensions.Tiles()
	if d.Separated {
		tiles *= len(d.Fields)
	}
	return tiles
}

// Get the total number of bytes that will be occupied in the file by this layer's header.
func (d *Layer) HeaderSize(h *PixiHeader) int {
	headerSize := 4 + 4                   // 4 bytes each for configuration and compression
	headerSize += 2 + len([]byte(d.Name)) // 2 bytes for name length, then name
	headerSize += 4                       // four bytes for dimension count
	for _, d := range d.Dimensions {
		headerSize += d.HeaderSize(h) // add each dimension header size
	}
	headerSize += 4 // four bytes for field count
	for _, f := range d.Fields {
		headerSize += f.HeaderSize(h) // add each field header size
	}
	headerSize += d.DiskTiles() * h.OffsetSize // offset size bytes for each real disk tile size in bytes
	headerSize += d.DiskTiles() * h.OffsetSize // offset size bytes for each tile offset
	headerSize += h.OffsetSize                 // offset size bytes for the next layer start offset
	return headerSize
}

// The on-disk size in bytes of the (potentially compressed) data set. Does not include the dataset
// header size.
func (d *Layer) DataSize() int64 {
	size := int64(0)
	for _, b := range d.TileBytes {
		size += b
	}
	return size
}

// Writes the binary description of the layer to the given stream, according to the specification
// in the Pixi header h.
func (d *Layer) WriteHeader(w io.Writer, h *PixiHeader) error {
	tiles := d.DiskTiles()
	if tiles != len(d.TileBytes) {
		return FormatError("invalid TileBytes: must have same number of elements as tiles in data set for valid pixi files")
	}
	if tiles != len(d.TileOffsets) {
		return FormatError("invalid TileOffsets: must have same number of elements as tiles in data set for valid pixi files")
	}

	// write configuration and compression
	configuration := uint32(0)
	if d.Separated {
		configuration = 1
	}
	err := h.Write(w, configuration)
	if err != nil {
		return err
	}

	err = h.Write(w, d.Compression)
	if err != nil {
		return err
	}

	// write layer name
	err = h.WriteFriendly(w, d.Name)
	if err != nil {
		return err
	}

	// write dimensions
	err = h.Write(w, uint32(len(d.Dimensions)))
	if err != nil {
		return err
	}
	for _, dim := range d.Dimensions {
		err = dim.Write(w, h)
		if err != nil {
			return err
		}
	}

	// write fields
	err = h.Write(w, uint32(len(d.Fields)))
	if err != nil {
		return err
	}
	for _, field := range d.Fields {
		err = field.Write(w, h)
		if err != nil {
			return err
		}
	}

	// write tile bytes, offsets, and start of next layer
	err = h.WriteOffsets(w, d.TileBytes)
	if err != nil {
		return err
	}
	err = h.WriteOffsets(w, d.TileOffsets)
	if err != nil {
		return err
	}
	err = h.WriteOffset(w, d.NextLayerStart)
	if err != nil {
		return err
	}

	return nil
}

// Reads a description of the layer from the given binary stream, according to the specification
// in the Pixi header h.
func (d *Layer) ReadLayer(r io.Reader, h *PixiHeader) error {
	// read configuration and compression
	var configuration uint32
	err := h.Read(r, &configuration)
	if err != nil {
		return err
	}
	d.Separated = configuration != 0
	err = h.Read(r, &d.Compression)
	if err != nil {
		return err
	}

	// read layer name
	d.Name, err = h.ReadFriendly(r)
	if err != nil {
		return err
	}

	// read dimensions
	var dimCount uint32
	err = h.Read(r, &dimCount)
	if err != nil {
		return err
	}
	if dimCount < 1 {
		return FormatError("must have at least one dimension for a valid pixi file")
	}
	d.Dimensions = make(DimensionSet, dimCount)
	for dInd := range d.Dimensions {
		dim := &Dimension{}
		err = dim.Read(r, h)
		if err != nil {
			return err
		}
		d.Dimensions[dInd] = dim
	}

	// read field types
	var fieldCount uint32
	err = h.Read(r, &fieldCount)
	if err != nil {
		return err
	}
	if fieldCount < 1 {
		return FormatError("must have at least one field for a valid pixi file")
	}
	d.Fields = make(FieldSet, fieldCount)
	for fInd := range d.Fields {
		field := &Field{}
		err = field.Read(r, h)
		if err != nil {
			return err
		}
		d.Fields[fInd] = field
	}

	// read tile bytes, offsets, and next layer start
	tiles := d.DiskTiles()
	d.TileBytes = make([]int64, tiles)
	err = h.ReadOffsets(r, d.TileBytes)
	if err != nil {
		return err
	}
	d.TileOffsets = make([]int64, tiles)
	err = h.ReadOffsets(r, d.TileOffsets)
	if err != nil {
		return err
	}
	d.NextLayerStart, err = h.ReadOffset(r)
	if err != nil {
		return err
	}

	return nil
}

// For a layer header which has already been written to the given position, writes the layer header again
// to the same location before returning the stream cursor to the position it was at previously. Generally
// this is used to update tile byte counts and tile offsets after they've been written to a stream.
func (l *Layer) OverwriteHeader(w io.WriteSeeker, h *PixiHeader, headerStartOffset int64) error {
	oldPos, err := w.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	_, err = w.Seek(headerStartOffset, io.SeekStart)
	if err != nil {
		return err
	}

	err = l.WriteHeader(w, h)
	if err != nil {
		return err
	}

	_, err = w.Seek(oldPos, io.SeekStart)
	return err
}

// Write the encoded tile data to the current stream position, updating the offset and byte count
// for this tile in the layer header (but not writing those offsets to the stream just yet). The
// data is written with a 4-byte checksum directly after it, which is used to verify data integrity
// when reading the tile later.
func (l *Layer) WriteTile(w io.WriteSeeker, h *PixiHeader, tileIndex int, data []byte) error {
	streamOffset, err := w.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	l.TileOffsets[tileIndex] = streamOffset

	writeAmt, err := l.Compression.WriteChunk(w, data)
	if err != nil {
		return err
	}
	l.TileBytes[tileIndex] = int64(writeAmt)

	checksum := crc32.ChecksumIEEE(data)
	return h.Write(w, checksum)
}

func (l *Layer) OverwriteTile(w io.WriteSeeker, h *PixiHeader, tileIndex int, data []byte) error {
	if l.TileOffsets[tileIndex] == 0 {
		panic("cannot overwrite a tile that has not already been written")
	}

	_, err := w.Seek(l.TileOffsets[tileIndex], io.SeekStart)
	if err != nil {
		return err
	}

	return l.WriteTile(w, h, tileIndex, data)
}

// Read a raw tile (not yet decoded into sample fields) at the given tile index. The tile must
// have been previously written (either in this session or a previous one) for this operation to succeed.
// The data is verified for integrity using a four-byte checksum placed directly after the saved
// tile data, and an error is returned (along with the data read into the chunk) if the checksum
// check fails.
func (l *Layer) ReadTile(r io.ReadSeeker, h *PixiHeader, tileIndex int, data []byte) error {
	if l.TileBytes[tileIndex] == 0 {
		panic("invalid tile byte count, likely tried to read a tile that hasn't been written yet")
	}

	_, err := r.Seek(l.TileOffsets[tileIndex], io.SeekStart)
	if err != nil {
		return err
	}

	_, err = l.Compression.ReadChunk(r, data)
	if err != nil {
		return err
	}

	// because compression can read more than necessary, we seek to tile start plus tile size
	// to get to the correct position for checksum
	_, err = r.Seek(l.TileOffsets[tileIndex]+l.TileBytes[tileIndex], io.SeekStart)
	if err != nil {
		return err
	}

	var savedChecksum uint32
	err = h.Read(r, &savedChecksum)
	if err != nil {
		return err
	}

	if savedChecksum != crc32.ChecksumIEEE(data) {
		return IntegrityError{TileIndex: tileIndex, LayerName: l.Name}
	}
	return nil
}
