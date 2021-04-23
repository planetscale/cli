package org

import (
	"errors"
	"os"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"

	"github.com/spf13/cobra"
)

func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Display the currently active organization",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := config.ProjectConfigPath()
			if err != nil {
				return err
			}

			cfg, err := ch.ConfigFS.NewFileConfig(configPath)
			if os.IsNotExist(err) {
				configPath, err = config.DefaultConfigPath()
				if err != nil {
					return err
				}

				cfg, err = ch.ConfigFS.NewFileConfig(configPath)
				if os.IsNotExist(err) {
					return errors.New("not authenticated, please authenticate with: \"pscale auth login\"")
				}

				if err != nil {
					return err
				}
			}

			if err != nil {
				return err
			}

			if cfg.Organization == "" {
				return errors.New("config file exists, but organization is not set")
			}

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("%s (from file: %s)\n", printer.Bold(cfg.Organization), configPath)
				return nil
			}

			if ch.Printer.Format() == printer.CSV {
				var res = []struct {
					Org string `json:"org"`
				}{{Org: cfg.Organization}}

				return ch.Printer.PrintResource(res)
			}

			return ch.Printer.PrintResource(map[string]string{"org": cfg.Organization})
		},
	}

	return cmd
}
