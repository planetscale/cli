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
		enable_auto_apply  bool
		disable_auto_apply bool
		autoApply          string // deprecated
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

			if flags.enable_auto_apply && flags.disable_auto_apply {
				return fmt.Errorf("cannot use both --enable-auto-apply and --disable-auto-apply flags together")
			}

			hasNewFlags := flags.enable_auto_apply || flags.disable_auto_apply
			hasDeprecatedFlag := flags.autoApply != ""

			if !hasNewFlags && !hasDeprecatedFlag {
				return fmt.Errorf("must specify either --enable-auto-apply, --disable-auto-apply, or --auto-apply")
			}

			if hasDeprecatedFlag {
				switch flags.autoApply {
				case "enable", "disable":
				default:
					return fmt.Errorf("--auto-apply accepts only \"enable\" or \"disable\" but got %q", flags.autoApply)
				}
			}

			var enable bool
			if hasNewFlags {
				enable = flags.enable_auto_apply
			} else {
				enable = flags.autoApply == "enable"
			}

			dr, err := client.DeployRequests.AutoApplyDeploy(ctx, &planetscale.AutoApplyDeployRequestRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Number:       n,
				Enable:       enable,
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
				ch.Printer.Printf("Successfully updated auto-apply changes for '%s/%s'.\n",
					printer.BoldBlue(database),
					printer.BoldBlue(dr.Number))
				return nil
			}

			return ch.Printer.PrintResource(toDeployRequest(dr))
		},
	}

	cmd.Flags().BoolVar(&flags.enable_auto_apply, "enable-auto-apply", false, "Enable auto-apply. The deploy request will automatically swap over to the new schema once ready.")
	cmd.Flags().BoolVar(&flags.disable_auto_apply, "disable-auto-apply", false, "Disable auto-apply. The deploy request will wait for your confirmation before swapping to the new schema. Use 'deploy-request apply' to apply the changes manually.")

	cmd.Flags().StringVar(&flags.autoApply, "auto-apply", "", "Update the auto apply setting for a deploy request. Possible values: [enable,disable]")
	cmd.Flags().MarkDeprecated("auto-apply", "use --enable-auto-apply or --disable-auto-apply instead")

	return cmd
}
