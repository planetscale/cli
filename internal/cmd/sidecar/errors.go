package sidecar

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func handleError(err error, database, branch, sidecar string) error {
	if cmdutil.ErrCode(err) != ps.ErrNotFound {
		return cmdutil.HandleError(err)
	}

	if sidecar != "" {
		return fmt.Errorf("sidecar %s does not exist in branch %s of database %s",
			printer.BoldBlue(sidecar), printer.BoldBlue(branch), printer.BoldBlue(database))
	}

	return fmt.Errorf("database %s or branch %s does not exist, or the branch is not a Neki branch",
		printer.BoldBlue(database), printer.BoldBlue(branch))
}
