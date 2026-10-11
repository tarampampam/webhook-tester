package web_test

import (
	"io/fs"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
	"gh.tarampamp.am/webhook-tester/v3/web"
)

func TestDist(t *testing.T) {
	t.Parallel()

	for _, fileSystem := range []fs.FS{
		web.Dist(true),
		web.Dist(false),
	} {
		f, err := fileSystem.Open("index.html")
		assert.NoError(t, err)
		assert.NotNil(t, f)

		// file is not empty
		bytes, err := f.Read(make([]byte, 2))
		assert.NoError(t, err)
		assert.Equal(t, 2, bytes)

		assert.NoError(t, f.Close())
	}
}
