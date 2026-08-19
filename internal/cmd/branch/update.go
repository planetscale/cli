package branch

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"

	"github.com/spf13/cobra"
)

// UpdateCmd updates configurable branch settings.
func UpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		newName           string
		deletionProtected bool
	}

	cmd := &cobra.Command{
		Use:   "update <database> <branch>",
		Short: "Update a branch's name or deletion protection",
		Long: `Update a branch's name or deletion protection.

Only the flags you pass are sent. At least one flag is required.`,
		Example: `  pscale branch update mydb main --new-name trunk
  pscale branch update mydb main --deletion-protected
  pscale branch update mydb main --deletion-protected=false`,
		Args: cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			newNameSet := cmd.Flags().Changed("new-name")
			protectedSet := cmd.Flags().Changed("deletion-protected")
			if !newNameSet && !protectedSet {
				return fmt.Errorf("at least one of --new-name or --deletion-protected must be provided")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			db, err := client.Databases.Get(ctx, &ps.GetDatabaseRequest{
				Organization: ch.Config.Organization,
				Database:     database,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("database %s does not exist in organization %s",
						printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating branch %s in %s",
				printer.BoldBlue(branch), printer.BoldBlue(database)))
			defer end()

			if db.Kind == ps.DatabaseEngineMySQL {
				updateReq := &ps.UpdateDatabaseBranchRequest{
					Organization: ch.Config.Organization,
					Database:     database,
					Branch:       branch,
				}
				if newNameSet {
					updateReq.NewName = flags.newName
				}
				if protectedSet {
					updateReq.DeletionProtected = &flags.deletionProtected
				}

				b, err := client.DatabaseBranches.Update(ctx, updateReq)
				if err != nil {
					return handleBranchUpdateError(ch, err, database, branch)
				}
				end()

				return printUpdatedBranch(ch, database, branch, flags.newName, newNameSet, ToDatabaseBranch(b))
			}

			updateReq := &ps.UpdatePostgresBranchRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
			}
			if newNameSet {
				updateReq.NewName = flags.newName
			}
			if protectedSet {
				updateReq.DeletionProtected = &flags.deletionProtected
			}

			b, err := client.PostgresBranches.Update(ctx, updateReq)
			if err != nil {
				return handleBranchUpdateError(ch, err, database, branch)
			}
			end()

			return printUpdatedBranch(ch, database, branch, flags.newName, newNameSet, ToPostgresBranch(b))
		},
	}

	cmd.Flags().StringVar(&flags.newName, "new-name", "", "Rename the branch")
	cmd.Flags().BoolVar(&flags.deletionProtected, "deletion-protected", false,
		"Protect the branch from deletion (use --deletion-protected=false to disable)")

	return cmd
}

func handleBranchUpdateError(ch *cmdutil.Helper, err error, database, branch string) error {
	switch cmdutil.ErrCode(err) {
	case ps.ErrNotFound:
		return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
			printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
	default:
		return cmdutil.HandleError(err)
	}
}

func printUpdatedBranch(ch *cmdutil.Helper, database, branch, newName string, renamed bool, resource any) error {
	if ch.Printer.Format() != printer.Human {
		return ch.Printer.PrintResource(resource)
	}

	if renamed {
		ch.Printer.Printf("Branch %s in %s was successfully updated and is now named %s.\n",
			printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(newName))
		return nil
	}

	ch.Printer.Printf("Branch %s in %s was successfully updated.\n",
		printer.BoldBlue(branch), printer.BoldBlue(database))
	return nil
}
