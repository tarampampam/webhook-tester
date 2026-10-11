package frontend

import (
	"bytes"
	_ "embed"
	"io"
	"io/fs"
	"time"
)

//go:embed fallback404.html
var fallback404html []byte

// fallback404 is a simple implementation of [fs.File] for serving the fallback 404 page.
type fallback404 struct{ rdr *bytes.Reader }

// newFallback404 creates a new instance of fallback404 with the embedded 404 page content.
func newFallback404() *fallback404 { return &fallback404{rdr: bytes.NewReader(fallback404html)} }

var ( // compile-time interface assertions
	_ fs.File     = (*fallback404)(nil)
	_ io.ReaderAt = (*fallback404)(nil)
	_ io.Seeker   = (*fallback404)(nil)
)

func (f *fallback404) Stat() (fs.FileInfo, error)       { return f, nil }
func (f *fallback404) Read(p []byte) (n int, err error) { return f.rdr.Read(p) }
func (f *fallback404) Close() error                     { return nil }

func (f *fallback404) ReadAt(p []byte, off int64) (n int, err error) {
	return f.rdr.ReadAt(p, off)
}

func (f *fallback404) Seek(offset int64, whence int) (int64, error) {
	return f.rdr.Seek(offset, whence)
}

func (f *fallback404) Name() string       { return "fallback404.html" }
func (f *fallback404) Size() int64        { return int64(len(fallback404html)) }
func (f *fallback404) Mode() fs.FileMode  { return 0o444 } //nolint:mnd // r--r--r--
func (f *fallback404) ModTime() time.Time { return time.Time{} }
func (f *fallback404) IsDir() bool        { return false }
func (f *fallback404) Sys() any           { return nil }
