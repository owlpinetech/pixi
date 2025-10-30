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

func TestFieldSetIndex(t *testing.T) {
	tests := []struct {
		name      string
		fields    FieldSet
		fieldName string
		wantIndex int
	}{
		{
			name:      "Field exists at index 0",
			fields:    FieldSet{{Name: "field1", Type: FieldInt8}, {Name: "field2", Type: FieldInt16}},
			fieldName: "field1",
			wantIndex: 0,
		},
		{
			name:      "Field exists at index 1",
			fields:    FieldSet{{Name: "field1", Type: FieldInt8}, {Name: "field2", Type: FieldInt16}},
			fieldName: "field2",
			wantIndex: 1,
		},
		{
			name:      "Field does not exist",
			fields:    FieldSet{{Name: "field1", Type: FieldInt8}, {Name: "field2", Type: FieldInt16}},
			fieldName: "field3",
			wantIndex: -1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotIndex := test.fields.Index(test.fieldName)
			if gotIndex != test.wantIndex {
				t.Errorf("fields.Index(%q) = %d, want %d", test.fieldName, gotIndex, test.wantIndex)
			}
		})
	}
}

func TestFieldSetOffset(t *testing.T) {
	tests := []struct {
		name       string
		fields     FieldSet
		fieldIndex int
		wantOffset int
	}{
		{
			name:       "Offset of first field",
			fields:     FieldSet{{Name: "field1", Type: FieldInt8}, {Name: "field2", Type: FieldInt16}},
			fieldIndex: 0,
			wantOffset: 0,
		},
		{
			name:       "Offset of second field",
			fields:     FieldSet{{Name: "field1", Type: FieldInt8}, {Name: "field2", Type: FieldInt16}},
			fieldIndex: 1,
			wantOffset: 1, // size of int8
		},
		{
			name:       "Offset of third field",
			fields:     FieldSet{{Name: "field1", Type: FieldInt8}, {Name: "field2", Type: FieldInt16}, {Name: "field3", Type: FieldFloat32}},
			fieldIndex: 2,
			wantOffset: 3, // size of int8 + size of int16
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotOffset := test.fields.Offset(test.fieldIndex)
			if gotOffset != test.wantOffset {
				t.Errorf("fields.Offset(%d) = %d, want %d", test.fieldIndex, gotOffset, test.wantOffset)
			}
		})
	}
}

func TestFieldSetNamedOffset(t *testing.T) {
	tests := []struct {
		name       string
		fields     FieldSet
		fieldName  string
		wantOffset int
		wantPanic  bool
	}{
		{
			name:       "Offset of first field",
			fields:     FieldSet{{Name: "field1", Type: FieldInt8}, {Name: "field2", Type: FieldInt16}},
			fieldName:  "field1",
			wantOffset: 0,
			wantPanic:  false,
		},
		{
			name:       "Offset of second field",
			fields:     FieldSet{{Name: "field1", Type: FieldInt8}, {Name: "field2", Type: FieldInt16}},
			fieldName:  "field2",
			wantOffset: 1, // size of int8
			wantPanic:  false,
		},
		{
			name:       "Field does not exist",
			fields:     FieldSet{{Name: "field1", Type: FieldInt8}, {Name: "field2", Type: FieldInt16}},
			fieldName:  "field3",
			wantOffset: 0,
			wantPanic:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !test.wantPanic {
						t.Errorf("did not expect to panic for test")
					}
				}
			}()

			gotOffset := test.fields.NamedOffset(test.fieldName)
			if gotOffset != test.wantOffset {
				t.Errorf("fields.NamedOffset(%q) = %d, want %d", test.fieldName, gotOffset, test.wantOffset)
			}
		})
	}
}
