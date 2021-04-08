package backup

import (
	"context"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	createReq := &ps.CreateBackupRequest{}
	cmd := &cobra.Command{
		Use:     "create <database> <branch> [options]",
		Short:   "Backup a branch's data and schema",
		Args:    cmdutil.RequiredArgs("database", "branch"),
		Aliases: []string{"b"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			database := args[0]
			branch := args[1]

			createReq.Database = database
			createReq.Branch = branch
			createReq.Organization = ch.Config.Organization

			client, err := ch.Config.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating backup of %s...", printer.BoldBlue(branch)))
			defer end()

			bkp, err := client.Backups.Create(ctx, createReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				case planetscale.ErrResponseMalformed:

					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Backup %s was successfully created!\n", printer.BoldBlue(bkp.Name))
				return nil
			}

			return ch.Printer.PrintResource(bkp)
		},
	}

	return cmd
}
