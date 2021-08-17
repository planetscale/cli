package password

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	createReq := &ps.DatabaseBranchPasswordRequest{}
	cmd := &cobra.Command{
		Use:     "create <database> <branch> <name>",
		Short:   "create password to access a branch's data",
		Args:    cmdutil.RequiredArgs("database", "branch", "name"),
		Aliases: []string{"p"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]
			name := args[2]
			createReq.Database = database
			createReq.Branch = branch
			createReq.Organization = ch.Config.Organization
			createReq.DisplayName = name

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating password of %s...", printer.BoldBlue(branch)))
			defer end()

			pass, err := client.Passwords.Create(ctx, createReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()
			if ch.Printer.Format() == printer.Human {
				saveWarning := printer.BoldRed("Please save the values below as they will not be shown again")
				ch.Printer.Printf("Password %s was successfully created.\n%s\n\n", printer.BoldBlue(pass.Name), saveWarning)
			}

			return ch.Printer.PrintResource(toPasswordWithPlainText(pass))
		},
	}

	return cmd
}
