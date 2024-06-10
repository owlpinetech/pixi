package pixi

import (
	"io"
	"os"
	"reflect"
	"testing"
)

func TestWriteCompressTile(t *testing.T) {
	tests := []struct {
		name        string
		compression Compression
		data        []byte
		expected    []byte
	}{
		{
			name:        "small data with flate compression",
			compression: CompressionFlate,
			data:        []byte{1, 2, 3, 4, 5},
			expected:    []byte{98, 100, 98, 102, 97, 5, 4, 0, 0, 255, 255},
		},
		{
			name:        "large data with none compression",
			compression: CompressionNone,
			data:        []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			expected:    []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := NewBuffer(10)
			under := Summary{
				Separated:   false,
				Compression: tc.compression,
				Dimensions:  []Dimension{{Size: int64(len(tc.data)), TileSize: int32(len(tc.data))}},
				Fields:      []Field{{Name: "byte", Type: FieldUint8}},
			}
			dataset, err := NewAppendDataset(under, buf, 10)
			if err != nil {
				t.Fatal(err)
			}
			err = dataset.writeTile(tc.data, 0)
			if err != nil {
				t.Errorf("expected no error, but got %s", err)
			}

			if !reflect.DeepEqual(tc.expected, buf.Bytes()[dataset.DiskDataStart():]) {
				t.Errorf("expected written to be %v, but got %v", tc.expected, buf.Bytes())
			}
			if !reflect.DeepEqual(int(dataset.TileBytes[0]), len(tc.expected)) {
				t.Errorf("expected TileBytes to be %d, but got %d", len(tc.expected), dataset.TileBytes[0])
			}
		})
	}
}

func TestAppendDatasetEvict(t *testing.T) {
	readCache := make(map[uint]*AppendTile)
	appendDataset := &AppendDataset{ReadCache: readCache}

	// Test that evict returns nil when cache is empty
	if err := appendDataset.evict(); err != nil {
		t.Errorf("expected evict to return nil, got %v", err)
	}

	// Test that evict removes the first element from the cache
	appendDataset.ReadCache = map[uint]*AppendTile{1: {}, 2: {}, 3: {}}
	if err := appendDataset.evict(); err != nil {
		t.Errorf("expected evict to return nil, got %v", err)
	}
	if len(appendDataset.ReadCache) != 2 {
		t.Errorf("expected ReadCache to have length 2 after evict, has length %d", len(appendDataset.ReadCache))
	}
	if err := appendDataset.evict(); err != nil {
		t.Errorf("expected evict to return nil, got %v", err)
	}
	if len(appendDataset.ReadCache) != 1 {
		t.Errorf("expected ReadCache to have length 1 after second evict, has length %d", len(appendDataset.ReadCache))
	}
}

func TestAppendAddTileToCache(t *testing.T) {
	// Create a new AppendDataset with maxInCache set to 3
	d := &AppendDataset{MaxInCache: 3, ReadCache: make(map[uint]*AppendTile)}

	// Add tiles to the cache and verify they are added correctly
	for i := uint(0); i < 5; i++ {
		tileIndex := i
		tile := AppendTile{Data: []byte{byte(i)}}
		err := d.addTileToCache(tileIndex, tile.Data)
		if err != nil {
			t.Errorf("addTileToCache failed: %v", err)
			return
		}
	}

	// Verify that the cache has 3 tiles (including eviction of older tiles if needed)
	if len(d.ReadCache) != 3 {
		t.Errorf("Expected ReadCache length to be 3, got %d", len(d.ReadCache))
		return
	}

	// Add another tile and verify it replaces the oldest tile in the cache
	for i := uint(5); i < 8; i++ {
		tileIndex := i
		tile := AppendTile{Data: []byte{byte(i)}}
		err := d.addTileToCache(tileIndex, tile.Data)
		if err != nil {
			t.Errorf("addTileToCache failed: %v", err)
			return
		}
	}

	// Verify that the cache has 3 tiles (after evicting older tiles if needed)
	if len(d.ReadCache) != 3 {
		t.Errorf("Expected ReadCache length to be 3, got %d", len(d.ReadCache))
		return
	}
}

func TestAppendAllReadAllSample(t *testing.T) {
	testCases := []struct {
		name        string
		compression Compression
	}{
		{name: "no comp", compression: CompressionNone},
		{name: "comp flate", compression: CompressionFlate},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			temp, err := os.CreateTemp("", tc.name+".pixi")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(temp.Name())

			under := Summary{
				Separated:   false,
				Compression: tc.compression,
				Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
				Fields:      []Field{{Type: FieldFloat64}, {Type: FieldInt16}, {Type: FieldUint64}},
			}
			dataset, err := NewAppendDataset(under, temp, 2)
			if err != nil {
				t.Fatal(err)
			}

			ytiles := dataset.Dimensions[1].Tiles()
			xtiles := dataset.Dimensions[0].Tiles()
			for ytile := 0; ytile < ytiles; ytile++ {
				for xtile := 0; xtile < xtiles; xtile++ {
					for x := 0; x < int(dataset.Dimensions[0].TileSize); x++ {
						for y := 0; y < int(dataset.Dimensions[1].TileSize); y++ {
							xDimInd := uint(xtile*int(dataset.Dimensions[0].TileSize) + x)
							yDimInd := uint(ytile*int(dataset.Dimensions[1].TileSize) + y)
							err := dataset.SetSample([]uint{xDimInd, yDimInd}, []any{1.5 + float64(xDimInd), int16(-xDimInd), uint64(yDimInd)})
							if err != nil {
								t.Fatal(err)
							}
							if dataset.WritingTileIndex != uint(xtile)+uint(ytile)*2 {
								t.Errorf("expected %d,%d tile index to be %d, got %d", xtile, ytile, uint(xtile)+uint(ytile)*2, dataset.WritingTileIndex)
							}
						}
					}
				}
			}
			err = dataset.Finalize()
			if err != nil {
				t.Fatal(err)
			}

			for x := 0; x < int(dataset.Dimensions[0].Size); x++ {
				for y := 0; y < int(dataset.Dimensions[1].Size); y++ {
					val, err := dataset.GetSample([]uint{uint(x), uint(y)})
					if err != nil {
						t.Fatalf("failed to get sample (%d,%d): %s", x, y, err)
					}
					if val[0].(float64) != 1.5+float64(x) {
						t.Errorf("expected first sample field at %d,%d to be %v, got %v", x, y, 1.5+float64(x), val[0])
					}
					if val[1].(int16) != int16(-x) {
						t.Errorf("expected second sample field at %d,%d to be %v, got %v", x, y, int16(-x), val[1])
					}
					if val[2].(uint64) != uint64(y) {
						t.Errorf("expected third sample field at %d,%d to be %v, got %v", x, y, uint64(y), val[2])
					}
					if len(dataset.ReadCache) > int(dataset.MaxInCache) {
						t.Errorf("expected read cache length to be less than %d, got %d", dataset.MaxInCache, len(dataset.ReadCache))
					}
				}
			}

			temp.Seek(0, io.SeekStart)
			rdSummary, err := ReadSummary(temp)
			if err != nil {
				t.Fatal(err)
			}

			rdDataset, err := ReadAppend(temp, rdSummary, 2)
			if err != nil {
				t.Fatal(err)
			}

			for x := 0; x < int(dataset.Dimensions[0].Size); x++ {
				for y := 0; y < int(dataset.Dimensions[1].Size); y++ {
					val, err := rdDataset.GetSample([]uint{uint(x), uint(y)})
					if err != nil {
						t.Fatalf("failed to get sample (%d,%d): %s", x, y, err)
					}
					if val[0].(float64) != 1.5+float64(x) {
						t.Errorf("expected first sample field at %d,%d to be %v, got %v", x, y, 1.5+float64(x), val[0])
					}
					if val[1].(int16) != int16(-x) {
						t.Errorf("expected second sample field at %d,%d to be %v, got %v", x, y, int16(-x), val[1])
					}
					if val[2].(uint64) != uint64(y) {
						t.Errorf("expected third sample field at %d,%d to be %v, got %v", x, y, uint64(y), val[2])
					}
					if len(rdDataset.ReadCache) > int(rdDataset.MaxInCache) {
						t.Errorf("expected read cache length to be less than %d, got %d", rdDataset.MaxInCache, len(rdDataset.ReadCache))
					}
				}
			}
		})
	}
}

func TestAppendAllReadAllSampleField(t *testing.T) {
	testCases := []struct {
		name        string
		separated   bool
		compression Compression
	}{
		//{name: "sep_no_comp", separated: true, compression: CompressionNone},
		//{name: "sep_comp_flate", separated: true, compression: CompressionFlate},
		//{name: "no_sep_no_comp", separated: false, compression: CompressionNone},
		{name: "no_sep_comp_flate", separated: false, compression: CompressionFlate},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			temp, err := os.CreateTemp("", tc.name+".pixi")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(temp.Name())

			under := Summary{
				Separated:   tc.separated,
				Compression: tc.compression,
				Dimensions:  []Dimension{{Size: 250, TileSize: 50}, {Size: 250, TileSize: 50}},
				Fields:      []Field{{Type: FieldFloat64}, {Type: FieldInt16}, {Type: FieldUint64}},
			}
			dataset, err := NewAppendDataset(under, temp, 2)
			if err != nil {
				t.Fatal(err)
			}

			ytiles := dataset.Dimensions[1].Tiles()
			xtiles := dataset.Dimensions[0].Tiles()
			if tc.separated {
				for ytile := 0; ytile < ytiles; ytile++ {
					for xtile := 0; xtile < xtiles; xtile++ {
						for x := 0; x < int(dataset.Dimensions[0].TileSize); x++ {
							for y := 0; y < int(dataset.Dimensions[1].TileSize); y++ {
								xDimInd := uint(xtile*int(dataset.Dimensions[0].TileSize) + x)
								yDimInd := uint(ytile*int(dataset.Dimensions[1].TileSize) + y)
								err := dataset.SetSampleField([]uint{xDimInd, yDimInd}, 0, 1.5+float64(xDimInd))
								if err != nil {
									t.Fatal(err)
								}
							}
						}
					}
				}
				for ytile := 0; ytile < ytiles; ytile++ {
					for xtile := 0; xtile < xtiles; xtile++ {
						for x := 0; x < int(dataset.Dimensions[0].TileSize); x++ {
							for y := 0; y < int(dataset.Dimensions[1].TileSize); y++ {
								xDimInd := uint(xtile*int(dataset.Dimensions[0].TileSize) + x)
								yDimInd := uint(ytile*int(dataset.Dimensions[1].TileSize) + y)
								err = dataset.SetSampleField([]uint{xDimInd, yDimInd}, 1, int16(-xDimInd))
								if err != nil {
									t.Fatal(err)
								}
							}
						}
					}
				}
				for ytile := 0; ytile < ytiles; ytile++ {
					for xtile := 0; xtile < xtiles; xtile++ {
						for x := 0; x < int(dataset.Dimensions[0].TileSize); x++ {
							for y := 0; y < int(dataset.Dimensions[1].TileSize); y++ {
								xDimInd := uint(xtile*int(dataset.Dimensions[0].TileSize) + x)
								yDimInd := uint(ytile*int(dataset.Dimensions[1].TileSize) + y)
								err = dataset.SetSampleField([]uint{xDimInd, yDimInd}, 2, uint64(yDimInd))
								if err != nil {
									t.Fatal(err)
								}
							}
						}
					}
				}
			} else {
				for ytile := 0; ytile < ytiles; ytile++ {
					for xtile := 0; xtile < xtiles; xtile++ {
						for x := 0; x < int(dataset.Dimensions[0].TileSize); x++ {
							for y := 0; y < int(dataset.Dimensions[1].TileSize); y++ {
								xDimInd := uint(xtile*int(dataset.Dimensions[0].TileSize) + x)
								yDimInd := uint(ytile*int(dataset.Dimensions[1].TileSize) + y)
								err := dataset.SetSampleField([]uint{xDimInd, yDimInd}, 0, 1.5+float64(xDimInd))
								if err != nil {
									t.Fatal(err)
								}
								err = dataset.SetSampleField([]uint{xDimInd, yDimInd}, 1, int16(-xDimInd))
								if err != nil {
									t.Fatal(err)
								}
								err = dataset.SetSampleField([]uint{xDimInd, yDimInd}, 2, uint64(yDimInd))
								if err != nil {
									t.Fatal(err)
								}
							}
						}
					}
				}
			}

			err = dataset.Finalize()
			if err != nil {
				t.Fatal(err)
			}

			for x := 0; x < int(dataset.Dimensions[0].Size); x++ {
				for y := 0; y < int(dataset.Dimensions[1].Size); y++ {
					val0, err := dataset.GetSampleField([]uint{uint(x), uint(y)}, 0)
					if err != nil {
						t.Fatalf("failed to get sample (%d,%d) 0: %s", x, y, err)
					}
					if val0.(float64) != 1.5+float64(x) {
						t.Fatalf("expected first sample field at %d,%d to be %v, got %v", x, y, 1.5+float64(x), val0)
					}
					val1, err := dataset.GetSampleField([]uint{uint(x), uint(y)}, 1)
					if err != nil {
						t.Fatalf("failed to get sample (%d,%d) 1: %s", x, y, err)
					}
					if val1.(int16) != int16(-x) {
						t.Fatalf("expected second sample field at %d,%d to be %v, got %v", x, y, int16(-x), val1)
					}
					val2, err := dataset.GetSampleField([]uint{uint(x), uint(y)}, 2)
					if err != nil {
						t.Fatalf("failed to get sample (%d,%d) 2: %s", x, y, err)
					}
					if val2.(uint64) != uint64(y) {
						t.Fatalf("expected third sample field at %d,%d to be %v, got %v", x, y, uint64(y), val2)
					}
					if len(dataset.ReadCache) > int(dataset.MaxInCache) {
						t.Fatalf("expected read cache length to be less than %d, got %d", dataset.MaxInCache, len(dataset.ReadCache))
					}
				}
			}

			temp.Seek(0, io.SeekStart)
			rdSummary, err := ReadSummary(temp)
			if err != nil {
				t.Fatal(err)
			}

			rdDataset, err := ReadAppend(temp, rdSummary, 2)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(rdSummary.TileBytes, dataset.Summary.TileBytes) {
				t.Errorf("expected tile bytes %v to equal %v\n", rdSummary.TileBytes, dataset.Summary.TileBytes)
			}

			for x := 0; x < int(dataset.Dimensions[0].Size); x++ {
				for y := 0; y < int(dataset.Dimensions[1].Size); y++ {
					val0, err := rdDataset.GetSampleField([]uint{uint(x), uint(y)}, 0)
					if err != nil {
						t.Fatalf("failed to get sample (%d,%d) 0: %s", x, y, err)
					}
					val1, err := rdDataset.GetSampleField([]uint{uint(x), uint(y)}, 1)
					if err != nil {
						t.Fatalf("failed to get sample (%d,%d) 1: %s", x, y, err)
					}
					val2, err := rdDataset.GetSampleField([]uint{uint(x), uint(y)}, 2)
					if err != nil {
						t.Fatalf("failed to get sample (%d,%d) 2: %s", x, y, err)
					}
					if val0.(float64) != 1.5+float64(x) {
						t.Fatalf("expected first sample field at %d,%d to be %v, got %v", x, y, 1.5+float64(x), val0)
					}
					if val1.(int16) != int16(-x) {
						t.Fatalf("expected second sample field at %d,%d to be %v, got %v", x, y, int16(-x), val1)
					}
					if val2.(uint64) != uint64(y) {
						t.Fatalf("expected third sample field at %d,%d to be %v, got %v", x, y, uint64(y), val2)
					}
					if len(rdDataset.ReadCache) > int(rdDataset.MaxInCache) {
						t.Fatalf("expected read cache length to be less than %d, got %d", rdDataset.MaxInCache, len(rdDataset.ReadCache))
					}
				}
			}
		})
	}
}
