package main

import (
	"os"

	"github.com/planetscale/cli/internal/cmd"
)

var (
	version string
	commit  string
	date    string
)

func main() {
	if err := cmd.Execute(version, commit, date); err != nil {
		// we don't print the error, because cobra does it for us
		os.Exit(1)
	}
}
