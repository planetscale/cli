package branch

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// RoutingRulesCmd is the top-level command for fetching or updating the routing rules of a branch.
func RoutingRulesCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "routing-rules <command>",
		Short: "Fetch or update your keyspace routing rules",
	}

	cmd.AddCommand(GetRoutingRulesCmd(ch))
	cmd.AddCommand(UpdateRoutingRulesCmd(ch))

	return cmd
}

// GetRoutingRulesCmd is the command for showing the routing rules of a branch.
func GetRoutingRulesCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <database> <branch>",
		Short: "Show the routing rules of a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			routingRules, err := client.DatabaseBranches.RoutingRules(ctx, &planetscale.BranchRoutingRulesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			if ch.Printer.Format() != printer.Human {
				return ch.Printer.PrintResource(routingRules)
			}

			err = ch.Printer.PrettyPrintJSON([]byte(routingRules.Raw))
			if err != nil {
				return fmt.Errorf("reading routingRules raw: %s", err)
			}

			return nil
		},
	}

	return cmd
}

// UpdateRoutingRulesCmd is the command for updating the routing rules of a branch.
func UpdateRoutingRulesCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		routingRules string
	}

	cmd := &cobra.Command{
		Use:   "update <database> <branch> --routing-rules <file>",
		Short: "Update the routing rules of a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			var data string

			if flags.routingRules != "" {
				rawRoutingRules, err := os.ReadFile(flags.routingRules)
				if err != nil {
					return err
				}
				data = string(rawRoutingRules)
			} else {
				stdinFile, err := os.Stdin.Stat()
				if err != nil {
					return err
				}

				if (stdinFile.Mode() & os.ModeCharDevice) == 0 {
					stdin, err := io.ReadAll(os.Stdin)
					if err != nil {
						return err
					}

					data = string(stdin)
				}
			}

			if len(data) == 0 {
				return errors.New("no routing rules provided, use the --routing-rules and provide a file or pipe the rules to standard in")
			}

			routingRules, err := client.DatabaseBranches.UpdateRoutingRules(ctx, &planetscale.UpdateBranchRoutingRulesRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				RoutingRules: data,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			if ch.Printer.Format() != printer.Human {
				return ch.Printer.PrintResource(routingRules)
			}

			err = ch.Printer.PrettyPrintJSON([]byte(routingRules.Raw))
			if err != nil {
				return fmt.Errorf("reading routing rules raw: %s", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.routingRules, "routing-rules", "", "The routing to set")

	return cmd
}
