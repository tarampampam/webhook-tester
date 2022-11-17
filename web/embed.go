//go:build !watch

package web

import (
	"embed"
	"io/fs"
)

//go:embed dist
var content embed.FS

// Content returns the embedded web content.
func Content() fs.FS {
	data, _ := fs.Sub(content, "dist")

	return data
}
