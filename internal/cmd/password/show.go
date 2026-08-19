package password

import (
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"

	ps "github.com/planetscale/cli/internal/planetscale"

	"github.com/spf13/cobra"
)

// ShowCmd shows the metadata for a branch password. The password secret is only
// available when the password is created or renewed, so it is never shown here.
func ShowCmd(ch *cmdutil.Helper) *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "show <database> <branch> [<password-id>]",
		Short: "Show a branch password",
		Long: `Show a branch password.

Only metadata is returned. The password itself is shown once, when it is
created or renewed, and cannot be retrieved afterwards.`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]

			if err := passwordSelector(args, name); err != nil {
				return err
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			var argID string
			if len(args) == 3 {
				argID = args[2]
			}

			passwordId, err := resolvePasswordID(ctx, ch, client, database, branch, argID, name)
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching password %s from %s/%s",
				printer.BoldBlue(passwordId), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			password, err := client.Passwords.Get(ctx, &ps.GetDatabaseBranchPasswordRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				PasswordId:   passwordId,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("password %s does not exist in branch %s of %s (organization: %s)",
						printer.BoldBlue(passwordId), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if err := ch.Printer.PrintResource(toPassword(password)); err != nil {
				return err
			}

			if ch.Printer.Format() == printer.Human && len(password.CIDRs) > 0 {
				ch.Printer.Printf("\nIP restrictions: %s\n", strings.Join(password.CIDRs, ", "))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Show password by name instead of ID")
	return cmd
}
