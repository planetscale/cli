package database

import (
	"context"
	"fmt"
	"net/url"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"

	"github.com/planetscale/planetscale-go/planetscale"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

// CreateCmd is the command for creating a database.
func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	createReq := &ps.CreateDatabaseRequest{}

	cmd := &cobra.Command{
		Use:   "create <database>",
		Short: "Create a database instance",
		Args:  cmdutil.RequiredArgs("database"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			createReq.Organization = ch.Config.Organization
			createReq.Name = args[0]

			if web {
				ch.Printer.Println("🌐  Redirecting you to create a database in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("%s/%s?name=%s&notes=%s&showDialog=true", cmdutil.ApplicationURL, ch.Config.Organization, url.QueryEscape(createReq.Name), url.QueryEscape(createReq.Notes)))
				if err != nil {
					return err
				}
				return nil
			}

			client, err := ch.Config.NewClientFromConfig()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress("Creating database...")
			defer end()
			database, err := client.Databases.Create(ctx, createReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("organization %s does not exist\n", printer.BoldBlue(ch.Config.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Database %s was successfully created!\n", printer.BoldBlue(database.Name))
				return nil
			}

			return ch.Printer.PrintResource(toDatabase(database))
		},
	}

	cmd.Flags().StringVar(&createReq.Notes, "notes", "", "notes for the database")
	cmd.Flags().BoolP("web", "w", false, "Create a database in your web browser")

	return cmd
}
