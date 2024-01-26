package password

import (
	"fmt"
	"strconv"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

func RenewCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "renew <database> <branch> <password-id>",
		Short: "Renew a branch password",
		Args:  cmdutil.RequiredArgs("database", "branch", "password-id"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]
			passwordId := args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Renewing password %s from %s/%s",
				printer.BoldBlue(passwordId), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			pass, err := client.Passwords.Renew(ctx, &ps.RenewDatabaseBranchPasswordRequest{
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

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Password %s was successfully renewed from %s.\n",
					printer.BoldBlue(passwordId), printer.BoldBlue(branch))
			}

			return ch.Printer.PrintResource(
				map[string]string{
					"result":      "password renewed",
					"password_id": passwordId,
					"username":    pass.Username,
					"branch":      branch,
					"ttl":         strconv.Itoa(pass.TTL),
				},
			)
		},
	}
	return cmd
}
