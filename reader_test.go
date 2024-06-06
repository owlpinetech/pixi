package pixi

import (
	"bytes"
	"reflect"
	"testing"
)

func FuzzWriteReadMetadata(f *testing.F) {
	f.Add("", "")
	f.Add("a", "b")
	f.Add("abcdefghijklnm", "opqrstuvwxyz")
	f.Fuzz(func(t *testing.T, key string, val string) {
		buf := new(bytes.Buffer)
		err := WriteMetadata(buf, key, val)
		if err != nil {
			t.Fatal(err)
		}
		outKey, outVal, err := ReadMetadata(buf)
		if err != nil {
			t.Fatal(err)
		}
		if key != outKey || val != outVal {
			t.Errorf("expected key %s, got %s, expected val %s, got %s", key, outKey, val, outVal)
		}
	})
}

func TestWriteReadDataSet(t *testing.T) {
	testCases := []struct {
		name string
		data DataSet
		err  error
	}{
		{
			name: "contig",
			data: DataSet{
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 4, TileSize: 4}, {Size: 4, TileSize: 2}, {Size: 3, TileSize: 3}},
				Fields:      []Field{{Name: "a", Type: FieldInt32}, {Name: "b", Type: FieldInt64}, {Name: "hello", Type: FieldInt16}},
				TileBytes:   []int64{100, 200},
				Offset:      8888,
			},
			err: nil,
		},
		{
			name: "sep",
			data: DataSet{
				Separated:   true,
				Compression: CompressionGzip,
				Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
				Fields:      []Field{{Name: "a", Type: FieldFloat64}, {Name: "hello", Type: FieldInt16}},
				TileBytes:   []int64{100, 200, 300, 400, 500, 600, 700, 800},
				Offset:      9898,
			},
			err: nil,
		},
		{
			name: "err",
			data: DataSet{
				Separated:   false,
				Compression: CompressionGzip,
				Dimensions:  []Dimension{{Size: 42, TileSize: 12}, {Size: 28, TileSize: 10}},
				Fields:      []Field{{Name: "a", Type: FieldFloat64}, {Name: "hello", Type: FieldInt16}},
				TileBytes:   []int64{100, 200, 300, 400, 500, 600, 700, 800, 900, 1000},
				Offset:      9898,
			},
			err: FormatError("TileBytes must have same number of tiles as data set for valid pixi files"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			err := WriteDataSet(buf, tc.data)
			if tc.err != nil {
				if err == nil {
					t.Fatalf("expected error %v but got none", tc.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			ds, err := ReadDataSet(buf)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(tc.data, ds) {
				t.Errorf("expected read dataset to be %v, got %v", tc.data, ds)
			}
		})
	}
}
