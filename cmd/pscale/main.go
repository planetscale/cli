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
	os.Exit(cmd.Execute(version, commit, date))
}
