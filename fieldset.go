package pixi

// An ordered set of named fields present in each sample of a layer in a Pixi file.
type FieldSet []Field

// The size in bytes of each sample in the data set. Each field has a fixed size, and a sample
// is made up of one element of each field, so the sample size is the sum of all field sizes.
func (set FieldSet) Size() int {
	sampleSize := 0
	for _, f := range set {
		sampleSize += f.Size()
	}
	return sampleSize
}

// The index of the (first) field with the given name in the set, or -1 if not found.
func (set FieldSet) Index(fieldName string) int {
	for i, field := range set {
		if field.Name == fieldName {
			return i
		}
	}
	return -1
}

// The byte offset of the field within a given sample. This is the sum of the sizes of all preceding fields.
func (set FieldSet) Offset(fieldIndex int) int {
	offset := 0
	for i := range fieldIndex {
		offset += set[i].Size()
	}
	return offset
}

// The byte offset of the field with the given name within a sample. This is the sum of the sizes of all preceding fields.
// Panics if the field is not found in the set.
func (set FieldSet) NamedOffset(fieldName string) int {
	offset := 0
	for _, field := range set {
		if field.Name == fieldName {
			return offset
		}
		offset += field.Size()
	}
	panic("pixi: field not found in field set")
}
