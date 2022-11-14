package fileserver

import (
	"io"
	"net/http"
	"os"
	"path"

	"github.com/labstack/echo/v4"
)

var fallback404 = []byte("<!doctype html><html><body><h1>Error 404</h1><h2>Not found</h2></body></html>") //nolint:lll,gochecknoglobals

func NewHandler(root http.FileSystem) func(c echo.Context) error {
	var (
		fileServer       = http.FileServer(root)
		errorPageContent []byte
	)

	if f, err := root.Open("404.html"); err == nil {
		errorPageContent, _ = io.ReadAll(f)
	}

	return func(c echo.Context) error {
		f, err := root.Open(path.Clean(c.Request().URL.Path))
		if os.IsNotExist(err) {
			if c.Request().Method == http.MethodHead {
				return c.NoContent(http.StatusNotFound)
			}

			if len(errorPageContent) > 0 {
				return c.HTMLBlob(http.StatusNotFound, errorPageContent)
			}

			return c.HTMLBlob(http.StatusNotFound, fallback404)
		}

		if err != nil { // looks like unneeded, but so looks better
			_ = f.Close()
		}

		if c.Request().Method == http.MethodHead {
			return c.NoContent(http.StatusOK)
		}

		return echo.WrapHandler(fileServer)(c)
	}
}
