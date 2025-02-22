package pixi

import "testing"

func TestFieldSetSize(t *testing.T) {
	tests := []struct {
		name     string
		fields   FieldSet
		wantSize int
	}{
		{
			name:     "No fields",
			fields:   FieldSet{},
			wantSize: 0,
		},
		{
			name:     "One field with size 1",
			fields:   FieldSet{{Name: "", Type: FieldInt8}},
			wantSize: 1,
		},
		{
			name:     "One field with size 2",
			fields:   FieldSet{{Name: "", Type: FieldInt16}},
			wantSize: 2,
		},
		{
			name:     "Multiple fields with different sizes",
			fields:   FieldSet{{Name: "", Type: FieldInt8}, {Name: "", Type: FieldFloat32}},
			wantSize: 5, // size of int8 + size of float32
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotSize := test.fields.Size()
			if gotSize != test.wantSize {
				t.Errorf("fields.Size() = %d, want %d", gotSize, test.wantSize)
			}
		})
	}
}
