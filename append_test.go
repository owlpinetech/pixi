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
			dataset, err := NewAppendDataset(false,
				tc.compression,
				[]Dimension{{Size: int64(len(tc.data)), TileSize: int32(len(tc.data))}},
				[]Field{{Name: "byte", Type: FieldUint8}},
				buf,
				10,
				0,
			)
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
