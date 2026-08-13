package deployrequest

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	"github.com/spf13/cobra"
)

// ThrottlerCmd groups deploy-request throttler commands.
func ThrottlerCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "throttler <command>",
		Short: "Show or update deploy request throttler configuration",
		Long:  "Show or update deploy request throttler configuration.\n\nThis is per deploy request, not the database-level throttler.",
	}

	cmd.AddCommand(ThrottlerShowCmd(ch))
	cmd.AddCommand(ThrottlerUpdateCmd(ch))
	return cmd
}

// ThrottlerShowCmd shows throttler configuration for a deploy request.
func ThrottlerShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database> <number>",
		Short: "Show throttler configuration for a deploy request",
		Args:  cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			numberStr := args[1]

			number, err := strconv.ParseUint(numberStr, 10, 64)
			if err != nil {
				return fmt.Errorf("the argument <number> is invalid: %s", err)
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching throttler config for deploy request %s/%s",
				printer.BoldBlue(database), printer.BoldBlue(number)))
			defer end()

			throttler, err := client.DeployRequests.GetThrottler(ctx, &planetscale.GetDeployRequestThrottlerRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Number:       number,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("deploy request '%s/%s' does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(number), printer.BoldBlue(ch.Config.Organization))
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

// ThrottlerUpdateCmd updates throttler configuration for a deploy request.
func ThrottlerUpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		ratio          int
		configurations []string
	}

	cmd := &cobra.Command{
		Use:   "update <database> <number>",
		Short: "Update throttler configuration for a deploy request",
		Long: `Update throttler configuration for a deploy request.

Use --ratio to apply one ratio (0-95) to all eligible keyspaces.
Use --configuration keyspace=ratio (repeatable) for per-keyspace ratios.
Pass exactly one of these modes. 0 effectively disables throttling; 95 slows migrations the most.`,
		Args: cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			numberStr := args[1]

			number, err := strconv.ParseUint(numberStr, 10, 64)
			if err != nil {
				return fmt.Errorf("the argument <number> is invalid: %s", err)
			}

			ratioSet := cmd.Flags().Changed("ratio")
			if !ratioSet && len(flags.configurations) == 0 {
				return fmt.Errorf("must specify --ratio or --configuration")
			}
			if ratioSet && len(flags.configurations) > 0 {
				return fmt.Errorf("cannot use both --ratio and --configuration; pick one mode")
			}

			updateReq := &planetscale.UpdateDeployRequestThrottlerRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Number:       number,
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

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating throttler config for deploy request %s/%s",
				printer.BoldBlue(database), printer.BoldBlue(number)))
			defer end()

			throttler, err := client.DeployRequests.UpdateThrottler(ctx, updateReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("deploy request '%s/%s' does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(number), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Updated throttler configuration for deploy request %s/%s.\n",
					printer.BoldBlue(database), printer.BoldBlue(number))
				return nil
			}

			return ch.Printer.PrintResource(toThrottler(throttler))
		},
	}

	cmd.Flags().IntVar(&flags.ratio, "ratio", 0, "Throttler ratio 0-95 applied to all eligible keyspaces")
	cmd.Flags().StringArrayVar(&flags.configurations, "configuration", nil, "Per-keyspace ratio as keyspace=ratio (repeatable)")

	return cmd
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

	orig *planetscale.DeployRequestThrottler
}

func (t *ThrottlerRow) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(t.orig, "", "  ")
}

func (t *ThrottlerRow) MarshalCSVValue() interface{} {
	return []*ThrottlerRow{t}
}

func toThrottler(throttler *planetscale.DeployRequestThrottler) *ThrottlerRow {
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
