package fileserver

import (
	"io"
	"net/http"
	"os"
	"path"

	"github.com/labstack/echo/v4"
)

var fallback404 = []byte("<html><body><h1>Error 404</h1><h2>Not found</h2></body></html>") //nolint:gochecknoglobals

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
			if len(errorPageContent) > 0 {
				return c.HTMLBlob(http.StatusNotFound, errorPageContent)
			}

			return c.HTMLBlob(http.StatusNotFound, fallback404)
		}

		if err != nil {
			_ = f.Close()
		}

		return echo.WrapHandler(fileServer)(c)
	}
}
