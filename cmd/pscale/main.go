package main

import (
	"fmt"
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
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
