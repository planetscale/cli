package branch

import (
	"errors"
	"fmt"
	"strings"

	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func SafeMigrationsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "safe-migrations <command>",
		Short: "Enable or disable safe migrations on a branch",
	}

	cmd.AddCommand(EnableSafeMigrationsCmd(ch))
	cmd.AddCommand(DisableSafeMigrationsCmd(ch))

	return cmd
}

func EnableSafeMigrationsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable <database> <branch>",
		Short: "Enable safe migrations for a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			db := args[0]
			branch := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Enabling safe migrations for %s", printer.BoldBlue(branch)))
			defer end()
			b, err := client.DatabaseBranches.EnableSafeMigrations(ctx, &ps.EnableSafeMigrationsRequest{
				Organization: ch.Config.Organization,
				Database:     db,
				Branch:       branch,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("branch %s does not exist in database %s", printer.BoldBlue(branch), printer.BoldBlue(db))
				case ps.ErrRetry:
					lintErrors, err := client.DatabaseBranches.LintSchema(ctx, &ps.LintSchemaRequest{
						Organization: ch.Config.Organization,
						Database:     db,
						Branch:       branch,
					})
					if err != nil {
						return cmdutil.HandleError(err)
					}

					if len(lintErrors) > 0 {
						if ch.Printer.Format() == printer.Human {
							var sb strings.Builder
							sb.WriteString(printer.Red("Enabling safe migrations failed. "))
							sb.WriteString("Fix the following errors and then try again:\n\n")
							for _, lintError := range lintErrors {
								fmt.Fprintf(&sb, "• %s\n", lintError.ErrorDescription)
							}

							return errors.New(sb.String())
						}

						return ch.Printer.PrintResource(toSchemaLintErrors(lintErrors))
					}

					return cmdutil.HandleError(err)
				default:
					return cmdutil.HandleError(err)
				}

			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Successfully enabled safe migrations for %s\n", printer.BoldBlue(branch))
				return nil
			} else {
				return ch.Printer.PrintResource(ToDatabaseBranch(b))
			}
		},
	}

	return cmd
}

func DisableSafeMigrationsCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable <database> <branch>",
		Short: "Disable safe migrations for a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			db := args[0]
			branch := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Disabling safe migrations for %s", printer.BoldBlue(branch)))
			defer end()
			b, err := client.DatabaseBranches.DisableSafeMigrations(ctx, &ps.DisableSafeMigrationsRequest{
				Organization: ch.Config.Organization,
				Database:     db,
				Branch:       branch,
			})
			if err != nil {
				return cmdutil.HandleError(err)
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Successfully disabled safe migrations for %s\n", printer.BoldBlue(branch))
				return nil
			} else {
				return ch.Printer.PrintResource(ToDatabaseBranch(b))
			}
		},
	}

	return cmd
}
