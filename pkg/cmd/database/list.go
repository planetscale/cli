package database

import (
	"github.com/planetscale/cli/config"
	"github.com/spf13/cobra"
)

// ListCmd is the command for listing all databases for an authenticated user.
func ListCmd(cfg *config.Config) *cobra.Command {
	// TODO(notfelineit/iheanyi): Add `--web` flag for opening the list of
	// databases here in the web UI.
	cmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO(iheanyi): Talk to API to list databases here and print them out.
			return nil
		},
	}

	return cmd
}
