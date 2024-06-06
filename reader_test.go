package pixi

import (
	"bytes"
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
