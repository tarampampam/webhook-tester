package web

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

// Generate mock distributive files, if needed.
//go:generate go run generate_dist_stub.go

//go:embed dist
var content embed.FS

// Dist returns frontend distributive files. If live is true, it returns files from the dist directory, otherwise
// from the embedded content. Live might be useful for development purposes.
func Dist(live bool) fs.FS {
	const distDirName = "dist"

	if live {
		// get the current file path (to resolve the dist directory path later)
		_, filePath, _, ok := runtime.Caller(0)
		if !ok {
			return noFs("unable to get the current file path")
		}

		return os.DirFS(path.Join(filepath.Dir(filePath), distDirName))
	} else {
		data, err := fs.Sub(content, distDirName)
		if err != nil {
			return noFs("dist directory not found")
		}

		return data
	}
}

// noFs is a mock fs.FS implementation, which returns an error on Open.
type noFs string

var _ fs.FS = (*noFs)(nil) // verify that noFs implements fs.FS

func (fs noFs) Open(string) (fs.File, error) { return nil, errors.New("web/dist: " + string(fs)) }
