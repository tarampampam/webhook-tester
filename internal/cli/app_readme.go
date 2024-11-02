//go:build generate

package main

import (
	"os"

	cliDocs "github.com/urfave/cli-docs/v3"

	"gh.tarampamp.am/webhook-tester/v2/internal/cli"
)

func main() {
	const readmePath = "../../README.md"

	if stat, err := os.Stat(readmePath); err == nil && stat.Mode().IsRegular() {
		if err = cliDocs.ToTabularToFileBetweenTags(cli.NewApp(), "app", readmePath); err != nil {
			panic(err)
		} else {
			println("✔ cli docs updated successfully")
		}
	} else if err != nil {
		println("⚠ readme file not found, cli docs not updated:", err.Error())
	}
}
