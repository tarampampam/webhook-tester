package j_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	j "gh.tarampamp.am/webhook-tester/v3/internal/httpserver/json"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestEncode(t *testing.T) {
	t.Parallel()

	t.Run("valid - common struct", func(t *testing.T) {
		t.Parallel()

		type payload struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		result, err := j.Encode(payload{Name: "Alice", Age: 30})
		assert.NoError(t, err)
		assert.JSONEq(t, `{"name": "Alice", "age": 30}`, string(result))
	})

	t.Run("valid - string", func(t *testing.T) {
		t.Parallel()

		result, err := j.Encode("hello")
		assert.NoError(t, err)
		assert.JSONEq(t, `"hello"`, string(result))
	})

	t.Run("valid - empty string", func(t *testing.T) {
		t.Parallel()

		result, err := j.Encode("")
		assert.NoError(t, err)
		assert.JSONEq(t, `""`, string(result))
	})

	t.Run("valid - nil pointer", func(t *testing.T) {
		t.Parallel()

		var p *struct{ Name string }

		result, err := j.Encode(p)
		assert.NoError(t, err)
		assert.JSONEq(t, `null`, string(result))
	})

	t.Run("invalid - unencodable value", func(t *testing.T) {
		t.Parallel()

		_, err := j.Encode(make(chan int))
		assert.ErrorContains(t, err, "encoding json:")
	})
}

func TestEncodeTo(t *testing.T) {
	t.Parallel()

	t.Run("valid - common struct", func(t *testing.T) {
		t.Parallel()

		type payload struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		var buf bytes.Buffer

		err := j.EncodeTo[payload](&buf, payload{Name: "Alice", Age: 30})
		assert.NoError(t, err)
		assert.JSONEq(t, `{"name": "Alice", "age": 30}`, buf.String())
	})

	t.Run("valid - string", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		err := j.EncodeTo[string](&buf, "hello")
		assert.NoError(t, err)
		assert.JSONEq(t, `"hello"`, buf.String())
	})

	t.Run("valid - appends newline", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		err := j.EncodeTo[string](&buf, "hello")
		assert.NoError(t, err)
		assert.True(t, strings.HasSuffix(buf.String(), "\n"))
	})

	t.Run("invalid - nil writer", func(t *testing.T) {
		t.Parallel()

		err := j.EncodeTo[struct{}](nil, struct{}{})
		assert.ErrorEqual(t, err, "nil writer")
	})

	t.Run("invalid - unencodable value", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer

		err := j.EncodeTo[chan int](&buf, make(chan int))
		assert.ErrorContains(t, err, "encoding json:")
	})

	t.Run("invalid - writer error", func(t *testing.T) {
		t.Parallel()

		fooErr := errors.New("foo error")

		err := j.EncodeTo[string](&faultyWriter{writeErr: fooErr}, "hello")
		assert.ErrorIs(t, err, fooErr)
		assert.ErrorContains(t, err, "encoding json:")
	})
}

type faultyWriter struct {
	writeErr error
}

func (f *faultyWriter) Write(_ []byte) (int, error) { return 0, f.writeErr }
