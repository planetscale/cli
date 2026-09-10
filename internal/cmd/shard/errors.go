package shard

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
)

func handleError(err error, database, branch, shardID, configurationProfile string) error {
	if cmdutil.ErrCode(err) != ps.ErrNotFound {
		return cmdutil.HandleError(err)
	}

	if shardID != "" {
		return fmt.Errorf("shard %s does not exist in branch %s of database %s",
			printer.BoldBlue(shardID), printer.BoldBlue(branch), printer.BoldBlue(database))
	}

	if configurationProfile != "" {
		return fmt.Errorf("database %s, branch %s, or configuration profile %s does not exist, or the branch is not a Neki branch",
			printer.BoldBlue(database), printer.BoldBlue(branch), printer.BoldBlue(configurationProfile))
	}

	return fmt.Errorf("database %s or branch %s does not exist, or the branch is not a Neki branch",
		printer.BoldBlue(database), printer.BoldBlue(branch))
}
