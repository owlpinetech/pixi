package pixi

// An ordered set of named fields present in each sample of a layer in a Pixi file.
type FieldSet []*Field

// The size in bytes of each sample in the data set. Each field has a fixed size, and a sample
// is made up of one element of each field, so the sample size is the sum of all field sizes.
func (set FieldSet) Size() int {
	sampleSize := 0
	for _, f := range set {
		sampleSize += f.Size()
	}
	return sampleSize
}
