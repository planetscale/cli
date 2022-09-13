package deployrequest

import (
	"fmt"
	"strconv"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

// EditCmd is the command for editing preferences on deploy requests.
func EditCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		autoApply string
	}

	cmd := &cobra.Command{
		Use:   "edit <database> <number> [flags]",
		Short: "Edit a deploy request",
		Args:  cmdutil.RequiredArgs("database", "number"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			number := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			n, err := strconv.ParseUint(number, 10, 64)
			if err != nil {
				return fmt.Errorf("the argument <number> is invalid: %s", err)
			}

			switch flags.autoApply {
			case "enabled", "disabled":
			default:
				return fmt.Errorf("--auto-apply accepts only \"enabled\" or \"disabled\" but got %q", flags.autoApply)
			}

			dr, err := client.DeployRequests.AutoApplyDeploy(ctx, &planetscale.AutoApplyDeployRequestRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Number:       n,
				Enable:       flags.autoApply == "enabled",
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

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Successfully enabled auto-apply changes for '%s/%s'.\n",
					printer.BoldBlue(database),
					printer.BoldBlue(dr.Number))
				return nil
			}

			return ch.Printer.PrintResource(toDeployRequest(dr))
		},
	}

	cmd.Flags().StringVar(&flags.autoApply, "auto-apply", "enabled", "Update the auto apply setting for a deploy request. Possible values: [enabled,disabled]")
	return cmd
}
