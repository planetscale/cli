package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	"github.com/spf13/cobra"
)

// ThrottlerCmd groups database-level Vitess migration throttler commands.
func ThrottlerCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "throttler <command>",
		Short: "Show or update database throttler configuration",
		Long: `Show or update database-level Vitess migration throttler configuration.

This sets the default throttler for future deploy requests on the database.
It is not the per-deploy-request throttler (pscale deploy-request throttler)
and not the tablet/vtctld throttler (pscale branch vtctld throttler).`,
	}

	cmd.AddCommand(ThrottlerShowCmd(ch))
	cmd.AddCommand(ThrottlerUpdateCmd(ch))
	return cmd
}

// ThrottlerShowCmd shows database-level throttler configuration.
func ThrottlerShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database>",
		Short: "Show database throttler configuration",
		Args:  cmdutil.RequiredArgs("database"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := requireVitessDatabase(ctx, ch, client, database); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching throttler config for database %s",
				printer.BoldBlue(database)))
			defer end()

			throttler, err := client.Databases.GetThrottler(ctx, &planetscale.GetDatabaseThrottlerRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			return ch.Printer.PrintResource(toThrottler(throttler))
		},
	}

	return cmd
}

// ThrottlerUpdateCmd updates database-level throttler configuration.
func ThrottlerUpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		ratio          int
		configurations []string
	}

	cmd := &cobra.Command{
		Use:   "update <database>",
		Short: "Update database throttler configuration",
		Long: `Update database-level Vitess migration throttler configuration.

Use --ratio to apply one ratio (0-95) to all eligible keyspaces.
Use --configuration keyspace=ratio (repeatable) for per-keyspace ratios.
Pass exactly one of these modes. 0 effectively disables throttling; 95 slows migrations the most.

This is the database default for future deploy requests, not a single deploy request.`,
		Args: cmdutil.RequiredArgs("database"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]

			ratioSet := cmd.Flags().Changed("ratio")
			if !ratioSet && len(flags.configurations) == 0 {
				return fmt.Errorf("must specify --ratio or --configuration")
			}
			if ratioSet && len(flags.configurations) > 0 {
				return fmt.Errorf("cannot use both --ratio and --configuration; pick one mode")
			}

			updateReq := &planetscale.UpdateDatabaseThrottlerRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			}

			if ratioSet {
				if flags.ratio < 0 || flags.ratio > 95 {
					return fmt.Errorf("--ratio must be between 0 and 95, got %d", flags.ratio)
				}
				updateReq.Ratio = &flags.ratio
			}

			if len(flags.configurations) > 0 {
				configs, err := parseThrottlerConfigurations(flags.configurations)
				if err != nil {
					return err
				}
				updateReq.Configurations = configs
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := requireVitessDatabase(ctx, ch, client, database); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating throttler config for database %s",
				printer.BoldBlue(database)))
			defer end()

			throttler, err := client.Databases.UpdateThrottler(ctx, updateReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Updated throttler configuration for database %s.\n",
					printer.BoldBlue(database))
				return nil
			}

			return ch.Printer.PrintResource(toThrottler(throttler))
		},
	}

	cmd.Flags().IntVar(&flags.ratio, "ratio", 0, "Throttler ratio 0-95 applied to all eligible keyspaces")
	cmd.Flags().StringArrayVar(&flags.configurations, "configuration", nil, "Per-keyspace ratio as keyspace=ratio (repeatable)")

	return cmd
}

func requireVitessDatabase(ctx context.Context, ch *cmdutil.Helper, client *planetscale.Client, database string) error {
	db, err := client.Databases.Get(ctx, &planetscale.GetDatabaseRequest{
		Organization: ch.Config.Organization,
		Database:     database,
	})
	if err != nil {
		switch cmdutil.ErrCode(err) {
		case planetscale.ErrNotFound:
			return fmt.Errorf("database %s does not exist in organization %s",
				printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
		default:
			return cmdutil.HandleError(err)
		}
	}
	if db.Kind != planetscale.DatabaseEngineMySQL {
		return fmt.Errorf("database throttler is only available for Vitess (MySQL) databases; %s is %s",
			printer.BoldBlue(database), printer.BoldBlue(string(db.Kind)))
	}
	return nil
}

func parseThrottlerConfigurations(values []string) ([]*planetscale.UpdateThrottlerConfiguration, error) {
	configs := make([]*planetscale.UpdateThrottlerConfiguration, 0, len(values))
	for _, value := range values {
		keyspace, ratioStr, ok := strings.Cut(value, "=")
		if !ok || keyspace == "" || ratioStr == "" {
			return nil, fmt.Errorf("invalid --configuration %q: expected keyspace=ratio", value)
		}
		ratio, err := strconv.Atoi(ratioStr)
		if err != nil {
			return nil, fmt.Errorf("invalid --configuration %q: ratio must be an integer", value)
		}
		if ratio < 0 || ratio > 95 {
			return nil, fmt.Errorf("invalid --configuration %q: ratio must be between 0 and 95", value)
		}
		configs = append(configs, &planetscale.UpdateThrottlerConfiguration{
			KeyspaceName: keyspace,
			Ratio:        ratio,
		})
	}
	return configs, nil
}

type ThrottlerRow struct {
	Keyspaces      string `header:"keyspaces" json:"keyspaces"`
	Configurations string `header:"configurations" json:"configurations"`

	orig *planetscale.DatabaseThrottler
}

func (t *ThrottlerRow) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(t.orig, "", "  ")
}

func (t *ThrottlerRow) MarshalCSVValue() interface{} {
	return []*ThrottlerRow{t}
}

func toThrottler(throttler *planetscale.DatabaseThrottler) *ThrottlerRow {
	configs := make([]string, 0, len(throttler.Configurations))
	for _, c := range throttler.Configurations {
		configs = append(configs, fmt.Sprintf("%s=%g", c.KeyspaceName, c.Ratio))
	}

	return &ThrottlerRow{
		Keyspaces:      strings.Join(throttler.Keyspaces, ", "),
		Configurations: strings.Join(configs, ", "),
		orig:           throttler,
	}
}
