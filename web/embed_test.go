package web_test

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"

	"gh.tarampamp.am/webhook-tester/v2/web"
)

func TestDist(t *testing.T) {
	t.Parallel()

	for _, fileSystem := range []fs.FS{
		web.Dist(true),
		web.Dist(false),
	} {
		f, err := fileSystem.Open("index.html")
		require.NoError(t, err)
		require.NotNil(t, f)

		// file is not empty
		bytes, err := f.Read(make([]byte, 2))
		require.NoError(t, err)
		require.Equal(t, 2, bytes)

		require.NoError(t, f.Close())
	}
}
