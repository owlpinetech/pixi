package pixi

// A sample is a list of field values, in field-index order, for at a single index / coordinate in a layer.
type Sample []any

// Creates a Sample from a map of named field values, according to the order of fields in the given layer.
func FromNamedSample(fieldset FieldSet, named map[string]any) Sample {
	sample := make(Sample, len(fieldset))
	for i, field := range fieldset {
		sample[i] = named[field.Name]
	}
	return sample
}

// Creates a map of named field values from the Sample, according to the order of fields in the given layer.
func (s Sample) Named(fieldSet FieldSet) map[string]any {
	named := make(map[string]any)
	for i, field := range fieldSet {
		named[field.Name] = s[i]
	}
	return named
}
