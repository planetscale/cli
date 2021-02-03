package org

import (
	"github.com/planetscale/cli/internal/config"

	"github.com/spf13/cobra"
)

func OrgCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use: "org <command>",
	}

	cmd.AddCommand(SwitchCmd(cfg))

	return cmd
}
