//go:build watch

package web

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// Content returns the web content using local file system.
//
//	---------------------------------------------------------
//	| ONLY FOR LOCAL DEVELOPMENT, NOT FOR PRODUCTION BUILD! |
//	---------------------------------------------------------
func Content() fs.FS {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("unable to get the current filename")
	}

	data, err := fs.Sub(os.DirFS(filepath.Dir(filename)+"/dist"), ".")
	if err != nil {
		panic(err)
	}

	return data
}
