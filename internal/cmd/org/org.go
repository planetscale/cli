package org

import (
	"github.com/planetscale/cli/internal/config"

	"github.com/spf13/cobra"
)

func OrgCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org <command>",
		Short: "Modify and manage organization options",
	}

	cmd.AddCommand(SwitchCmd(cfg))
	cmd.AddCommand(ShowCmd(cfg))
	cmd.AddCommand(ListCmd(cfg))

	return cmd
}
