package ft

import (
	"encoding/json"
	"strings"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

// ToJSON marshals the given data to JSON and returns the resulting byte slice.
func ToJSON(t *testing.T, data any) []byte {
	t.Helper()

	j, err := json.Marshal(data)
	assert.NoError(t, err)

	return j
}

// JSONPath extracts a value from JSON data by dot-notation path (e.g. "foo.bar.baz") and asserts it is of type TOut.
//
// Note: [json.Unmarshal] decodes all numbers as float64, so use JSONPath[float64] for numeric fields.
func JSONPath[TOut any, TData ~string | ~[]byte](t *testing.T, data TData, path string) TOut {
	t.Helper()

	var root any
	if err := json.Unmarshal([]byte(data), &root); err != nil {
		t.Fatalf("JSONPath: unmarshal: %v", err)
	}

	cur := root
	for key := range strings.SplitSeq(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("JSONPath: %q: expected object, got %T in %s", key, cur, string(data))
		}

		cur, ok = m[key]
		if !ok {
			t.Fatalf("JSONPath: %q: key not found in %s", key, string(data))
		}
	}

	v, ok := cur.(TOut)
	if !ok {
		var zero TOut
		t.Fatalf("JSONPath: %q: expected %T, got %T (%v) in %s", path, zero, cur, cur, string(data))
	}

	return v
}
