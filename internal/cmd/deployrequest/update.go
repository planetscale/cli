package deployrequest

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	"github.com/spf13/cobra"
)

// UpdateCmd updates deploy-request settings. `edit` is kept as an alias.
func UpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		enable_auto_apply          bool
		disable_auto_apply         bool
		autoApply                  string // deprecated
		enable_auto_delete_branch  bool
		disable_auto_delete_branch bool
	}

	cmd := &cobra.Command{
		Use:     "update <database> <number> [flags]",
		Aliases: []string{"edit"},
		Short:   "Update a deploy request",
		Long: `Update settings on a deploy request.

Use --enable-auto-apply / --disable-auto-apply to control gated cutover, and
--enable-auto-delete-branch / --disable-auto-delete-branch to control whether
the source branch is deleted after a successful deploy. At least one setting
must be passed; unset flags are not sent.`,
		Args: cmdutil.RequiredArgs("database", "number"),
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
			if flags.enable_auto_delete_branch && flags.disable_auto_delete_branch {
				return fmt.Errorf("cannot use both --enable-auto-delete-branch and --disable-auto-delete-branch flags together")
			}

			hasNewAutoApply := flags.enable_auto_apply || flags.disable_auto_apply
			hasDeprecatedFlag := flags.autoApply != ""
			hasAutoApply := hasNewAutoApply || hasDeprecatedFlag
			hasAutoDelete := flags.enable_auto_delete_branch || flags.disable_auto_delete_branch

			if !hasAutoApply && !hasAutoDelete {
				return fmt.Errorf("must specify at least one of --enable-auto-apply, --disable-auto-apply, --enable-auto-delete-branch, --disable-auto-delete-branch, or --auto-apply")
			}

			if hasDeprecatedFlag {
				switch flags.autoApply {
				case "enable", "disable":
				default:
					return fmt.Errorf("--auto-apply accepts only \"enable\" or \"disable\" but got %q", flags.autoApply)
				}
			}

			handleDRErr := func(err error) error {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("deploy request '%s/%s' does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(number), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			var dr *planetscale.DeployRequest

			if hasAutoApply {
				var enable bool
				if hasNewAutoApply {
					enable = flags.enable_auto_apply
				} else {
					enable = flags.autoApply == "enable"
				}

				dr, err = client.DeployRequests.AutoApplyDeploy(ctx, &planetscale.AutoApplyDeployRequestRequest{
					Organization: ch.Config.Organization,
					Database:     database,
					Number:       n,
					Enable:       enable,
				})
				if err != nil {
					return handleDRErr(err)
				}
			}

			if hasAutoDelete {
				dr, err = client.DeployRequests.AutoDeleteBranch(ctx, &planetscale.AutoDeleteBranchRequest{
					Organization: ch.Config.Organization,
					Database:     database,
					Number:       n,
					Enable:       flags.enable_auto_delete_branch,
				})
				if err != nil {
					return handleDRErr(err)
				}
			}

			if ch.Printer.Format() == printer.Human {
				updated := make([]string, 0, 2)
				if hasAutoApply {
					updated = append(updated, "auto-apply")
				}
				if hasAutoDelete {
					updated = append(updated, "auto-delete-branch")
				}
				ch.Printer.Printf("Successfully updated %s for '%s/%s'.\n",
					strings.Join(updated, " and "),
					printer.BoldBlue(database),
					printer.BoldBlue(dr.Number))
				return nil
			}

			return ch.Printer.PrintResource(toDeployRequest(dr))
		},
	}

	cmd.Flags().BoolVar(&flags.enable_auto_apply, "enable-auto-apply", false, "Enable auto-apply. The deploy request will automatically swap over to the new schema once ready.")
	cmd.Flags().BoolVar(&flags.disable_auto_apply, "disable-auto-apply", false, "Disable auto-apply. The deploy request will wait for your confirmation before swapping to the new schema. Use 'deploy-request apply' to apply the changes manually.")
	cmd.Flags().BoolVar(&flags.enable_auto_delete_branch, "enable-auto-delete-branch", false, "Delete the source branch after the deploy request completes.")
	cmd.Flags().BoolVar(&flags.disable_auto_delete_branch, "disable-auto-delete-branch", false, "Keep the source branch after the deploy request completes.")

	cmd.Flags().StringVar(&flags.autoApply, "auto-apply", "", "Update the auto apply setting for a deploy request. Possible values: [enable,disable]")
	cmd.Flags().MarkDeprecated("auto-apply", "use --enable-auto-apply or --disable-auto-apply instead")

	return cmd
}

// EditCmd is an alias of UpdateCmd so existing tests keep compiling.
func EditCmd(ch *cmdutil.Helper) *cobra.Command {
	return UpdateCmd(ch)
}
