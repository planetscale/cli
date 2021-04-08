package cmdutil

import (
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
)

// Helper is passed to every single command and is used by individual
// subcommands.
type Helper struct {
	// Config contains globally sourced configuration
	Config *config.Config

	// Printer is used to print output of a command to stdout.
	Printer *printer.Printer
}
