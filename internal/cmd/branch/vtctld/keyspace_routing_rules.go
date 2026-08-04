package vtctld

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/spf13/cobra"
)

func GetKeyspaceRoutingRulesCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-keyspace-routing-rules <database> <branch>",
		Short: "Get live keyspace routing rules for a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Fetching keyspace routing rules for %s…",
					progressTarget(ch.Config.Organization, database, branch)))
			defer end()

			data, err := client.Vtctld.GetKeyspaceRoutingRules(ctx, &ps.VtctldGetKeyspaceRoutingRulesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	return cmd
}

func ApplyKeyspaceRoutingRulesCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		rules       string
		rulesFile   string
		cells       []string
		skipRebuild bool
	}

	cmd := &cobra.Command{
		Use:   "apply-keyspace-routing-rules <database> <branch>",
		Short: "Replace live keyspace routing rules for a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			rulesSet := cmd.Flags().Changed("rules")
			rulesFileSet := cmd.Flags().Changed("rules-file")
			if rulesSet == rulesFileSet {
				return errors.New("must specify exactly one of --rules or --rules-file")
			}

			raw := []byte(flags.rules)
			if rulesFileSet {
				var err error
				raw, err = os.ReadFile(flags.rulesFile)
				if err != nil {
					return fmt.Errorf("reading keyspace routing rules: %w", err)
				}
			}

			var rules ps.VtctldKeyspaceRoutingRules
			if err := json.Unmarshal(raw, &rules); err != nil {
				return fmt.Errorf("parsing keyspace routing rules: %w", err)
			}
			if rules.Rules == nil {
				rules.Rules = []ps.VtctldKeyspaceRoutingRule{}
			}
			for _, rule := range rules.Rules {
				if rule.FromKeyspace == "" || rule.ToKeyspace == "" {
					return errors.New("each rule must include from_keyspace and to_keyspace")
				}
			}

			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(
				fmt.Sprintf("Applying keyspace routing rules for %s…",
					progressTarget(ch.Config.Organization, database, branch)))
			defer end()

			data, err := client.Vtctld.ApplyKeyspaceRoutingRules(ctx, &ps.VtctldApplyKeyspaceRoutingRulesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Rules:        rules.Rules,
				SkipRebuild:  flags.skipRebuild,
				RebuildCells: flags.cells,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()
			return ch.Printer.PrettyPrintJSON(data)
		},
	}

	cmd.Flags().StringVarP(&flags.rules, "rules", "r", "", "Keyspace routing rules as JSON")
	cmd.Flags().StringVarP(&flags.rulesFile, "rules-file", "f", "", "Path to keyspace routing rules JSON")
	cmd.Flags().StringSliceVarP(&flags.cells, "cells", "c", nil, "Limit SrvVSchema rebuilding to these cells")
	cmd.Flags().BoolVar(&flags.skipRebuild, "skip-rebuild", false, "Skip rebuilding SrvVSchema objects")

	return cmd
}
