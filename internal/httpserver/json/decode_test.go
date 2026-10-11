package j_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	j "gh.tarampamp.am/webhook-tester/v3/internal/httpserver/json"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestDecode(t *testing.T) {
	t.Parallel()

	t.Run("valid - common json", func(t *testing.T) {
		t.Parallel()

		type payload struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		result, err := j.Decode[payload](io.NopCloser(strings.NewReader(`{"name": "Alice", "age": 30}`)))
		assert.NoError(t, err)
		assert.DeepEqual(t, &payload{Name: "Alice", Age: 30}, result)
	})

	t.Run("valid - empty string", func(t *testing.T) {
		t.Parallel()

		result, err := j.Decode[string](io.NopCloser(strings.NewReader(`""`)))
		assert.NoError(t, err)
		assert.Equal(t, "", *result)
	})

	t.Run("invalid - empty body", func(t *testing.T) {
		t.Parallel()

		_, err := j.Decode[struct{}](io.NopCloser(strings.NewReader(``)))
		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("invalid - nil body", func(t *testing.T) {
		t.Parallel()

		_, err := j.Decode[struct{}](nil)
		assert.ErrorEqual(t, err, "nil request body")
		assert.NotErrorIs(t, err, j.ErrInvalidJSON)
	})

	t.Run("invalid - unknown fields", func(t *testing.T) {
		t.Parallel()

		_, err := j.Decode[struct {
			Name string `json:"name"`
		}](io.NopCloser(strings.NewReader(`{"name": "Alice", "age": 30}`)))

		assert.ErrorEqual(t, err, `invalid json: decoding json: unknown field "age"`)
		assert.ErrorIs(t, err, j.ErrInvalidJSON)
	})

	t.Run("invalid - trailing data", func(t *testing.T) {
		t.Parallel()

		_, err := j.Decode[struct {
			Name string `json:"name"`
		}](io.NopCloser(strings.NewReader(`{"name": "Alice"}{"name": "Bob"}`)))

		assert.ErrorEqual(t, err, "invalid json: unexpected extra JSON in body")
		assert.ErrorIs(t, err, j.ErrInvalidJSON)
	})

	t.Run("invalid - reject unexpected keys", func(t *testing.T) {
		t.Parallel()

		type payload struct {
			Name string `json:"name"`
		}

		_, err := j.Decode[payload](io.NopCloser(strings.NewReader(`{"name": "Alice", "unexpected": "value"}`)))
		assert.ErrorEqual(t, err, `invalid json: decoding json: unknown field "unexpected"`)
		assert.ErrorIs(t, err, j.ErrInvalidJSON)
	})

	t.Run("error on closing", func(t *testing.T) {
		t.Parallel()

		fooErr := errors.New("foo error")

		_, err := j.Decode[struct {
			Name string `json:"name"`
		}](&faultyCloser{
			Reader:   strings.NewReader(`{"foo": "bar"}`),
			closeErr: fooErr,
		})

		assert.ErrorIs(t, err, fooErr)
		assert.ErrorEqual(t, err, `invalid json: decoding json: unknown field "foo": closing: foo error`)
		assert.ErrorIs(t, err, j.ErrInvalidJSON)
	})
}

type faultyCloser struct {
	io.Reader

	closeErr error
}

func (f *faultyCloser) Close() error { return f.closeErr }
