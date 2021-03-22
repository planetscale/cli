package org

import (
	"errors"
	"fmt"

	"github.com/planetscale/cli/internal/config"

	"github.com/spf13/cobra"
)

func ShowCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Display the currently active organization",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.DefaultGlobalConfig()
			if err != nil {
				return err
			}

			if cfg.Organization == "" {
				return errors.New("config file exists, but organization is not set")
			}

			fmt.Println(cfg.Organization)

			return nil
		},
	}

	return cmd
}
