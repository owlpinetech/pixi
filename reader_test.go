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
		data Summary
		err  error
	}{
		{
			name: "contig",
			data: Summary{
				Metadata:    map[string]string{},
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 4, TileSize: 4}, {Size: 4, TileSize: 2}, {Size: 3, TileSize: 3}},
				Fields:      []Field{{Name: "a", Type: FieldInt32}, {Name: "b", Type: FieldInt64}, {Name: "hello", Type: FieldInt16}},
				TileBytes:   []int64{100, 200},
				TileOffsets: []int64{80, 160},
			},
			err: nil,
		},
		{
			name: "with metadata",
			data: Summary{
				Metadata:    map[string]string{"one": "two", "three": "four"},
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 4, TileSize: 4}, {Size: 4, TileSize: 2}, {Size: 3, TileSize: 3}},
				Fields:      []Field{{Name: "a", Type: FieldInt32}, {Name: "b", Type: FieldInt64}, {Name: "hello", Type: FieldInt16}},
				TileBytes:   []int64{100, 200},
				TileOffsets: []int64{70, 30},
			},
			err: nil,
		},
		{
			name: "no names",
			data: Summary{
				Metadata:    map[string]string{},
				Separated:   false,
				Compression: CompressionNone,
				Dimensions:  []Dimension{{Size: 4, TileSize: 4}, {Size: 4, TileSize: 2}, {Size: 3, TileSize: 3}},
				Fields:      []Field{{Type: FieldInt32}, {Type: FieldInt64}, {Type: FieldInt16}},
				TileBytes:   []int64{100, 200},
				TileOffsets: []int64{100, 200},
			},
			err: nil,
		},
		{
			name: "sep",
			data: Summary{
				Metadata:    map[string]string{},
				Separated:   true,
				Compression: CompressionFlate,
				Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
				Fields:      []Field{{Name: "a", Type: FieldFloat64}, {Name: "hello", Type: FieldInt16}},
				TileBytes:   []int64{100, 200, 300, 400, 500, 600, 700, 800},
				TileOffsets: []int64{100, 200, 300, 400, 500, 600, 700, 800},
			},
			err: nil,
		},
		{
			name: "tile bytes err",
			data: Summary{
				Metadata:    map[string]string{},
				Separated:   true,
				Compression: CompressionFlate,
				Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
				Fields:      []Field{{Name: "a", Type: FieldFloat64}, {Name: "hello", Type: FieldInt16}},
				TileBytes:   []int64{100, 200, 300, 400, 500, 600, 700},
				TileOffsets: []int64{100, 200, 300, 400, 500, 600, 700, 800},
			},
			err: FormatError("TileBytes must have same number of tiles as data set for valid pixi files"),
		},
		{
			name: "tile offsets err",
			data: Summary{
				Metadata:    map[string]string{},
				Separated:   true,
				Compression: CompressionFlate,
				Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
				Fields:      []Field{{Name: "a", Type: FieldFloat64}, {Name: "hello", Type: FieldInt16}},
				TileBytes:   []int64{100, 200, 300, 400, 500, 600, 700, 800},
				TileOffsets: []int64{100, 200, 300, 400, 500, 600, 700},
			},
			err: FormatError("TileOffsets must have same number of tiles as data set for valid pixi files"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := NewBuffer(10)
			err := WriteSummary(buf, tc.data)
			if tc.err != nil {
				if err == nil {
					t.Fatalf("expected error %v but got none", tc.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			amtWritten := len(buf.Bytes())
			if amtWritten != int(tc.data.DiskDataStart()) {
				t.Errorf("expected data start to be %d, got %d", amtWritten, tc.data.DiskDataStart())
			}

			readBuf := NewBufferFrom(buf.Bytes())
			ds, err := ReadSummary(readBuf)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(tc.data, ds) {
				t.Errorf("expected read dataset to be %v, got %v", tc.data, ds)
			}
		})
	}
}
