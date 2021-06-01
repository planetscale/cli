package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/planetscale/cli/internal/cmd"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
)

var (
	version string
	commit  string
	date    string
)

func main() {
	var format printer.Format
	if err := cmd.Execute(version, commit, date, &format); err != nil {
		switch format {
		case printer.JSON:
			fmt.Fprintf(os.Stderr, `{"error": "%s"}`, err)
		default:
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		}

		var cmdErr *cmdutil.Error
		if errors.As(err, &cmdErr) {
			os.Exit(cmdErr.ExitCode)
		} else {
			os.Exit(1)
		}
	}
}
