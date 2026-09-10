package admin

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func handleError(err error, database, branch string) error {
	if cmdutil.ErrCode(err) != ps.ErrNotFound {
		return cmdutil.HandleError(err)
	}

	return fmt.Errorf("database %s or branch %s does not exist, the branch is not a Neki branch, or it has no admin",
		printer.BoldBlue(database), printer.BoldBlue(branch))
}
