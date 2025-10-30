package pixi

import "testing"

func TestSampleFromNamedSample(t *testing.T) {
	fields := FieldSet{
		{Name: "field1", Type: FieldInt32},
		{Name: "field2", Type: FieldInt64},
		{Name: "field3", Type: FieldFloat32},
	}

	named := map[string]any{
		"field1": int32(42),
		"field2": int64(10000000000),
		"field3": float32(3.14),
		"field4": "extra field",
	}

	sample := FromNamedSample(fields, named)
	if sample[0] != int32(42) {
		t.Errorf("Expected field1 to be 42, got %v", sample[0])
	}
	if sample[1] != int64(10000000000) {
		t.Errorf("Expected field2 to be 10000000000, got %v", sample[1])
	}
	if sample[2] != float32(3.14) {
		t.Errorf("Expected field3 to be 3.14, got %v", sample[2])
	}
}

func TestSampleNamed(t *testing.T) {
	fields := FieldSet{
		{Name: "field1", Type: FieldInt32},
		{Name: "field2", Type: FieldInt64},
		{Name: "field3", Type: FieldFloat32},
	}

	sample := Sample{int32(42), int64(10000000000), float32(3.14)}

	named := sample.Named(fields)
	if named["field1"] != int32(42) {
		t.Errorf("Expected field1 to be 42, got %v", named["field1"])
	}
	if named["field2"] != int64(10000000000) {
		t.Errorf("Expected field2 to be 10000000000, got %v", named["field2"])
	}
	if named["field3"] != float32(3.14) {
		t.Errorf("Expected field3 to be 3.14, got %v", named["field3"])
	}
}
