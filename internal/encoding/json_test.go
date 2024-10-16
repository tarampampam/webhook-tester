package encoding_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gh.tarampamp.am/webhook-tester/v2/internal/encoding"
)

type someJSONStruct struct {
	String   string          `json:"string"`
	Int      int             `json:"int"`
	Uint     uint            `json:"uint"`
	Float    float64         `json:"float"`
	StrSlice []string        `json:"str_slice"`
	IntSlice []int           `json:"int_slice"`
	Nested   *someJSONStruct `json:"nested,omitempty"`
}

func TestJSON_Decode(t *testing.T) {
	t.Parallel()

	var v = someJSONStruct{
		String:   "string",
		Int:      1,
		Uint:     1,
		Float:    1.1,
		StrSlice: []string{"a", "b"},
		IntSlice: []int{1, 2},
		Nested: &someJSONStruct{
			String:   "string",
			Int:      1,
			Uint:     1,
			Float:    1.1,
			StrSlice: []string{"a", "b"},
			IntSlice: []int{1, 2},
		},
	}

	j, err := (encoding.JSON{}).Encode(v)
	require.NoError(t, err)

	assert.JSONEq(t,
		`{
"string":"string",
"int":1,
"uint":1,
"float":1.1,
"str_slice":["a","b"],
"int_slice":[1,2],
"nested":{
	"string":"string",
	"int":1,
	"uint":1,
	"float":1.1,
	"str_slice":["a","b"],
	"int_slice":[1,2]
}}`, string(j))
}

func TestJSON_Encode(t *testing.T) {
	t.Parallel()

	var v someJSONStruct

	err := (encoding.JSON{}).Decode([]byte(`{
"string":"string",
"int":1,
"uint":1,
"float":1.1,
"str_slice":["a","b"],
"int_slice":[1,2],
"nested":{
	"string":"string",
	"int":1,
	"uint":1,
	"float":1.1,
	"str_slice":["a","b"],
	"int_slice":[1,2]
}}`), &v)

	require.NoError(t, err)

	assert.Equal(t, v, someJSONStruct{
		String:   "string",
		Int:      1,
		Uint:     1,
		Float:    1.1,
		StrSlice: []string{"a", "b"},
		IntSlice: []int{1, 2},
		Nested: &someJSONStruct{
			String:   "string",
			Int:      1,
			Uint:     1,
			Float:    1.1,
			StrSlice: []string{"a", "b"},
			IntSlice: []int{1, 2},
		},
	})
}
