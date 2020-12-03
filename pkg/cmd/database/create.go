package database

import (
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

// CreateCmd is the command for creating a database.
func CreateCmd(cfg *config.Config) *cobra.Command {
	// TODO(iheanyi): Add flags [name].
	cmd := &cobra.Command{
		Use: "create",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO(iheanyi): Talk to API and create a database here. Preferably
			// abstracted away in some client.
			return nil
		},
	}

	return cmd
}
