package pixi

import (
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
			name:        "small data with gzip compression",
			compression: CompressionGzip,
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
			under := DataSet{
				Separated:   false,
				Compression: tc.compression,
				Dimensions:  []Dimension{{Size: int64(len(tc.data)), TileSize: int32(len(tc.data))}},
				Fields:      []Field{{Name: "byte", Type: FieldUint8}},
			}
			dataset, err := NewAppendDataset(under, buf, 10, 0)
			if err != nil {
				t.Fatal(err)
			}
			err = dataset.writeCompressTile(tc.data, 0)
			if err != nil {
				t.Errorf("expected no error, but got %s", err)
			}

			if !reflect.DeepEqual(tc.expected, buf.Bytes()) {
				t.Errorf("expected written to be %v, but got %v", tc.expected, buf.Bytes())
			}
			if !reflect.DeepEqual(int(dataset.TileBytes[0]), len(tc.expected)) {
				t.Errorf("expected TileBytes to be %d, but got %d", len(tc.expected), dataset.TileBytes[0])
			}
		})
	}
}

func TestAppendSetGetSeparatedSampleField(t *testing.T) {
	buf := NewBuffer(10)
	under := DataSet{
		Separated:   false,
		Compression: CompressionNone,
		Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
		Fields:      []Field{{Type: FieldFloat64}, {Type: FieldInt16}, {Type: FieldUint64}},
	}
	dataset, err := NewAppendDataset(under, buf, 10, 0)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name               string
		indices            []uint
		field              uint
		val                any
		expectedWriteIndex uint
	}{
		{
			name:               "first index, first field",
			indices:            []uint{0, 0},
			field:              0,
			val:                4.5,
			expectedWriteIndex: 0,
		},
		{
			name:               "first index, second field",
			indices:            []uint{0, 0},
			field:              1,
			val:                int16(-13),
			expectedWriteIndex: 0,
		},
		{
			name:               "first index, third field",
			indices:            []uint{0, 0},
			field:              2,
			val:                uint64(987654321),
			expectedWriteIndex: 0,
		},
		{
			name:               "third index, first field",
			indices:            []uint{0, 1},
			field:              0,
			val:                156.234,
			expectedWriteIndex: 0,
		},
		{
			name:               "second index, second field",
			indices:            []uint{1, 0},
			field:              1,
			val:                int16(-97),
			expectedWriteIndex: 0,
		},
		{
			name:               "fifth index, second field",
			indices:            []uint{3, 0},
			field:              1,
			val:                int16(1013),
			expectedWriteIndex: 1,
		},
	}

	for _, tc := range tests {
		err = dataset.SetSampleField(tc.indices, tc.field, tc.val)
		if err != nil {
			t.Fatalf("expected no error for SetSampleField, but got %s", err)
		}

		getVal, err := dataset.GetSampleField(tc.indices, tc.field)
		if err != nil {
			t.Fatalf("expected no error for GetSampleField, but got %s", err)
		}

		if getVal != tc.val {
			t.Errorf("expected to get %v after setting, got %v", tc.val, getVal)
		}
		if dataset.WritingTileIndex != tc.expectedWriteIndex {
			t.Errorf("expected writing tile index to be %d, got %d", tc.expectedWriteIndex, dataset.WritingTileIndex)
		}
	}
}

func TestAppendSetGetSeparatedSample(t *testing.T) {
	buf := NewBuffer(10)
	under := DataSet{
		Separated:   false,
		Compression: CompressionNone,
		Dimensions:  []Dimension{{Size: 4, TileSize: 2}, {Size: 4, TileSize: 2}},
		Fields:      []Field{{Type: FieldFloat64}, {Type: FieldInt16}, {Type: FieldUint64}},
	}
	dataset, err := NewAppendDataset(under, buf, 10, 0)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name               string
		indices            []uint
		sample             []any
		expectedWriteIndex uint
	}{
		{
			name:               "first index",
			indices:            []uint{0, 0},
			sample:             []any{4.5, int16(-13), uint64(987654321)},
			expectedWriteIndex: 0,
		},
		{
			name:               "third index",
			indices:            []uint{0, 1},
			sample:             []any{156.234, int16(78), uint64(0)},
			expectedWriteIndex: 0,
		},
		{
			name:               "second index",
			indices:            []uint{1, 0},
			sample:             []any{0.0001, int16(-97), uint64(11111)},
			expectedWriteIndex: 0,
		},
		{
			name:               "fifth index",
			indices:            []uint{3, 0},
			sample:             []any{18.03, int16(1013), uint64(02)},
			expectedWriteIndex: 1,
		},
	}

	for _, tc := range tests {
		err = dataset.SetSample(tc.indices, tc.sample)
		if err != nil {
			t.Fatalf("expected no error for SetSampleField, but got %s", err)
		}

		getVal, err := dataset.GetSample(tc.indices)
		if err != nil {
			t.Fatalf("expected no error for GetSampleField, but got %s", err)
		}

		if !reflect.DeepEqual(tc.sample, getVal) {
			t.Errorf("expected written to be %v, but got %v", tc.sample, getVal)
		}
		if dataset.WritingTileIndex != tc.expectedWriteIndex {
			t.Errorf("expected writing tile index to be %d, got %d", tc.expectedWriteIndex, dataset.WritingTileIndex)
		}
	}
}
