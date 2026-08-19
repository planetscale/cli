package database

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	"github.com/spf13/cobra"
)

// AggressiveCutoverCmd groups database-level aggressive cutover commands.
func AggressiveCutoverCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aggressive-cutover <command>",
		Short: "Show or change aggressive cutover for a database",
		Long: `Show, enable, or disable aggressive cutover for a Vitess database.

When enabled, future deploy requests on this database cut over more
aggressively when waiting on table locks. This is a database-level setting,
not the same as forcing cutover on a single deploy request
(pscale deploy-request force-cutover).

See https://planetscale.com/docs/vitess/schema-changes/aggressive-cutover`,
	}

	cmd.AddCommand(AggressiveCutoverShowCmd(ch))
	cmd.AddCommand(AggressiveCutoverEnableCmd(ch))
	cmd.AddCommand(AggressiveCutoverDisableCmd(ch))
	return cmd
}

// AggressiveCutoverShowCmd shows whether aggressive cutover is enabled.
func AggressiveCutoverShowCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <database>",
		Short: "Show whether aggressive cutover is enabled",
		Args:  cmdutil.RequiredArgs("database"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := requireVitessDatabase(ctx, ch, client, database, "aggressive cutover"); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching aggressive cutover status for %s",
				printer.BoldBlue(database)))
			defer end()

			status, err := client.Databases.GetAggressiveCutover(ctx, &planetscale.AggressiveCutoverRequest{
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

			return printAggressiveCutover(ch, database, status)
		},
	}

	return cmd
}

// AggressiveCutoverEnableCmd enables aggressive cutover for a database.
func AggressiveCutoverEnableCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable <database>",
		Short: "Enable aggressive cutover for a database",
		Args:  cmdutil.RequiredArgs("database"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := requireVitessDatabase(ctx, ch, client, database, "aggressive cutover"); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Enabling aggressive cutover for %s",
				printer.BoldBlue(database)))
			defer end()

			status, err := client.Databases.EnableAggressiveCutover(ctx, &planetscale.AggressiveCutoverRequest{
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

			return printAggressiveCutover(ch, database, status)
		},
	}

	return cmd
}

// AggressiveCutoverDisableCmd disables aggressive cutover for a database.
func AggressiveCutoverDisableCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable <database>",
		Short: "Disable aggressive cutover for a database",
		Args:  cmdutil.RequiredArgs("database"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if err := requireVitessDatabase(ctx, ch, client, database, "aggressive cutover"); err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Disabling aggressive cutover for %s",
				printer.BoldBlue(database)))
			defer end()

			status, err := client.Databases.DisableAggressiveCutover(ctx, &planetscale.AggressiveCutoverRequest{
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

			return printAggressiveCutover(ch, database, status)
		},
	}

	return cmd
}

type aggressiveCutoverView struct {
	Database string `header:"database" json:"database"`
	Enabled  bool   `header:"enabled" json:"enabled"`

	orig *planetscale.AggressiveCutover
}

func (a *aggressiveCutoverView) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(a.orig, "", "  ")
}

func (a *aggressiveCutoverView) MarshalCSVValue() interface{} {
	return []*aggressiveCutoverView{a}
}

func toAggressiveCutover(database string, status *planetscale.AggressiveCutover) *aggressiveCutoverView {
	return &aggressiveCutoverView{
		Database: database,
		Enabled:  status.Enabled,
		orig:     status,
	}
}

func printAggressiveCutover(ch *cmdutil.Helper, database string, status *planetscale.AggressiveCutover) error {
	view := toAggressiveCutover(database, status)
	if ch.Printer.Format() == printer.Human {
		state := "disabled"
		if status.Enabled {
			state = "enabled"
		}
		ch.Printer.Printf("Aggressive cutover is %s for %s\n", state, printer.BoldBlue(database))
		return nil
	}
	return ch.Printer.PrintResource(view)
}
