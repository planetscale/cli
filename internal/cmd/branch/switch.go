package branch

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

func SwitchCmd(ch *cmdutil.Helper) *cobra.Command {
	var parentBranch string
	var autoCreate bool

	cmd := &cobra.Command{
		Use:   "switch <branch>",
		Short: "Switches the current project to use the specified branch",
		Args:  cmdutil.RequiredArgs("branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			branch := args[0]

			client, err := ch.Config.NewClientFromConfig()
			if err != nil {
				return err
			}

			ch.Printer.Printf("Finding branch %s on database %s\n",
				printer.BoldBlue(branch), printer.BoldBlue(ch.Config.Database))

			_, err = client.DatabaseBranches.Get(ctx, &ps.GetDatabaseBranchRequest{
				Organization: ch.Config.Organization,
				Database:     ch.Config.Database,
				Branch:       branch,
			})
			if err != nil && !errorIsNotFound(err) {
				return err
			}

			if errorIsNotFound(err) {
				if !autoCreate {
					return errors.New("branch does not exist in specified database. Use --create to automatically create during switch")
				}

				end := ch.Printer.PrintProgress(fmt.Sprintf("Branch does not exist, creating %s branch from %s...", printer.BoldBlue(branch), printer.BoldBlue(parentBranch)))
				defer end()

				createReq := &ps.CreateDatabaseBranchRequest{
					Organization: ch.Config.Organization,
					Database:     ch.Config.Database,
					Name:         branch,
					ParentBranch: parentBranch,
				}

				_, err = client.DatabaseBranches.Create(ctx, createReq)
				if err != nil {
					return err
				}

				end()
			}

			cfg := config.FileConfig{
				Organization: ch.Config.Organization,
				Database:     ch.Config.Database,
				Branch:       branch,
			}

			if err := cfg.WriteProject(); err != nil {
				return errors.Wrap(err, "error writing project configuration file")
			}

			ch.Printer.Printf(
				"Successfully switched to branch %s on database %s",
				printer.BoldBlue(branch),
				printer.BoldBlue(ch.Config.Database),
			)

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization,
		"The organization for the current user")
	cmd.PersistentFlags().StringVar(&ch.Config.Database, "database", ch.Config.Database,
		"The database this project is using")
	cmd.Flags().StringVar(&parentBranch, "parent-branch", "main",
		"parent branch to inherit from if a new branch is being created")
	cmd.Flags().BoolVar(&autoCreate, "create", false,
		"if enabled, will automatically create the branch if it does not exist")

	cmd.MarkPersistentFlagRequired("database") // nolint:errcheck
	return cmd
}

func errorIsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == http.StatusText(http.StatusNotFound)
}
