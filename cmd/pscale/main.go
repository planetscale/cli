package main

import (
	"os"

	"github.com/planetscale/cli/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		// we don't print the error, because cobra does it for us
		os.Exit(1)
	}
}
