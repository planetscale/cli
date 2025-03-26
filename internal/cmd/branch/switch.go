package branch

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

func SwitchCmd(ch *cmdutil.Helper) *cobra.Command {
	var parentBranch string
	var autoCreate bool
	var wait bool

	cmd := &cobra.Command{
		Use:   "switch <branch>",
		Short: "Switches the current project to use the specified branch",
		Args:  cmdutil.RequiredArgs("branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			branch := args[0]

			client, err := ch.Client()
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
			errorIsNotFound := cmdutil.ErrCode(err) == ps.ErrNotFound

			if err != nil && !errorIsNotFound {
				return cmdutil.HandleError(err)
			}

			if errorIsNotFound {
				if !autoCreate {
					return fmt.Errorf("branch does not exist in database %s and organization %s. Use --create to automatically create during switch",
						ch.Config.Database, ch.Config.Organization)
				}

				end := ch.Printer.PrintProgress(fmt.Sprintf("Branch does not exist, creating %s branch...", printer.BoldBlue(branch)))
				defer end()

				createReq := &ps.CreateDatabaseBranchRequest{
					Organization: ch.Config.Organization,
					Database:     ch.Config.Database,
					Name:         branch,
					ParentBranch: parentBranch,
				}

				_, err = client.DatabaseBranches.Create(ctx, createReq)
				if err != nil {
					switch cmdutil.ErrCode(err) {
					case ps.ErrNotFound:
						return fmt.Errorf("database %s does not exist in organization %s",
							printer.BoldBlue(ch.Config.Database), printer.BoldBlue(ch.Config.Organization))
					default:
						return cmdutil.HandleError(err)
					}
				}
				end()

				// wait and check until the DB is ready
				if wait {
					end := ch.Printer.PrintProgress(fmt.Sprintf("Waiting until branch %s is ready...", printer.BoldBlue(branch)))
					defer end()
					getReq := &ps.GetDatabaseBranchRequest{
						Organization: ch.Config.Organization,
						Database:     ch.Config.Database,
						Branch:       branch,
					}
					if err := waitUntilReady(ctx, client, ch.Printer, ch.Debug(), getReq); err != nil {
						return err
					}
					end()
				}
			}

			cfg := config.FileConfig{
				Organization: ch.Config.Organization,
				Database:     ch.Config.Database,
				Branch:       branch,
			}

			if err := cfg.WriteProject(); err != nil {
				return fmt.Errorf("error writing project configuration file: %w", err)
			}

			ch.Printer.Printf(
				"Successfully switched to branch %s on database %s\n",
				printer.BoldBlue(branch),
				printer.BoldBlue(ch.Config.Database),
			)

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Database, "database", ch.Config.Database,
		"The database this project is using")
	cmd.Flags().StringVar(&parentBranch, "parent-branch", "",
		"parent branch to inherit from if a new branch is being created")
	cmd.Flags().BoolVar(&autoCreate, "create", false,
		"if enabled, will automatically create the branch if it does not exist")
	cmd.Flags().BoolVar(&wait, "wait", false, "wait until the branch is ready")

	cmd.MarkPersistentFlagRequired("database") // nolint:errcheck
	return cmd
}
