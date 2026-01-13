package role

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func ResetDefaultCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		force bool
	}

	cmd := &cobra.Command{
		Use:   "reset-default <database> <branch>",
		Short: "Reset the credentials for the default `postgres` role",
		Long:  "This command resets the credentials for the default `postgres` role in the database, allowing you to reconfigure access. Any connections using the `postgres` role will need to be updated with the new credentials.",
		Args:  cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			org := ch.Config.Organization
			database := args[0]
			branch := args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			if !flags.force {
				if ch.Printer.Format() != printer.Human {
					return fmt.Errorf("cannot delete password with the output format %q (run with --force to override)", ch.Printer.Format())
				}

				confirmationName := fmt.Sprintf("%s/%s", database, branch)
				if !printer.IsTTY {
					return fmt.Errorf("cannot confirm deletion of branch %q (run with --force to override)", confirmationName)
				}

				confirmationMessage := fmt.Sprintf("%s %s %s", printer.Bold("Please type"),
					printer.BoldBlue(confirmationName), printer.Bold("to confirm:"))

				prompt := &survey.Input{
					Message: confirmationMessage,
				}

				var userInput string
				err = survey.AskOne(prompt, &userInput)
				if err != nil {
					if err == terminal.InterruptErr {
						os.Exit(0)
					} else {
						return err
					}
				}

				// If the confirmations don't match up, let's return an error.
				if userInput != confirmationName {
					return errors.New("incorrect database and branch name entered, skipping reset")
				}

			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Resetting default postgres role for %s/%s...", printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			role, err := client.PostgresRoles.ResetDefaultRole(cmd.Context(), &ps.ResetDefaultRoleRequest{
				Organization: org,
				Database:     database,
				Branch:       branch,
			})
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
				saveWarning := printer.BoldRed("Please save the values below as they will not be shown again. We recommend using these credentials only for creating usernames and passwords for accessing your database.")

				ch.Printer.Printf("Role was successfully reset in %s/%s.\n%s\n\n", printer.BoldBlue(database), printer.BoldBlue(branch), saveWarning)
				printPostgresRoleCredentials(ch.Printer, toPostgresRole(role))
				return nil
			}

			return ch.Printer.PrintResource(toPostgresRole(role))
		},
	}

	cmd.Flags().BoolVar(&flags.force, "force", false, "Force reset without confirmation")

	return cmd
}

type PostgresRole struct {
	PublicID      string `header:"id" json:"id"`
	Name          string `header:"name" json:"name"`
	Username      string `header:"username" json:"username"`
	Password      string `header:"password" json:"password"`
	AccessHostURL string `header:"access_host_url" json:"access_host_url"`
	DatabaseURL   string `header:"database_url" json:"database_url"`

	orig *ps.PostgresRole
}

func toPostgresRole(role *ps.PostgresRole) *PostgresRole {
	return &PostgresRole{
		PublicID:      role.ID,
		Name:          role.Name,
		Username:      role.Username,
		Password:      role.Password,
		AccessHostURL: role.AccessHostURL,
		DatabaseURL:   buildPostgresConnectionURL(role.Username, role.Password, role.AccessHostURL),
		orig:          role,
	}
}

// printPostgresRoleCredentials prints role credentials in a vertical layout.
func printPostgresRoleCredentials(p *printer.Printer, role *PostgresRole) {
	p.Printf("%-17s  %s\n", "ID", role.PublicID)
	p.Printf("%-17s  %s\n", "NAME", role.Name)
	p.Printf("%-17s  %s\n", "USERNAME", role.Username)
	p.Printf("%-17s  %s\n", "PASSWORD", role.Password)
	p.Printf("%-17s  %s\n", "ACCESS HOST URL", role.AccessHostURL)
	p.Printf("%-17s  %s\n", "DATABASE URL", role.DatabaseURL)
}

// buildPostgresConnectionURL constructs a PostgreSQL connection URL from role credentials.
func buildPostgresConnectionURL(username, password, accessHostURL string) string {
	host, port, err := net.SplitHostPort(accessHostURL)
	if err != nil {
		// If no port specified, use the host as-is and default to 5432
		host = accessHostURL
		port = "5432"
	}

	return fmt.Sprintf("postgresql://%s:%s@%s:%s/postgres?sslmode=verify-full",
		url.PathEscape(username),
		url.PathEscape(password),
		host,
		port,
	)
}
