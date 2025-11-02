//go:build ignore

package main

import (
	"errors"
	"os"
	"path"
)

func main() {
	const distDir = "./dist"

	if _, err := os.Stat(distDir); err != nil && errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(distDir, 0755); err != nil {
			panic(err)
		}
	}

	var (
		indexPath  = path.Join(distDir, "index.html")
		robotsPath = path.Join(distDir, "robots.txt")
	)

	if _, err := os.Stat(indexPath); err != nil && errors.Is(err, os.ErrNotExist) {
		if err = os.WriteFile(indexPath, []byte("<html><!-- generated automatically --></html>\n"), 0644); err != nil {
			panic(err)
		}
	}

	if _, err := os.Stat(robotsPath); err != nil && errors.Is(err, os.ErrNotExist) {
		if err = os.WriteFile(robotsPath, []byte("User-agent: *\nDisallow: /\n"), 0644); err != nil {
			panic(err)
		}
	}
}
