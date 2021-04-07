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
				fmt.Println("🌐  Redirecting you to create a database in your web browser.")
				err := browser.OpenURL(fmt.Sprintf("%s/%s?name=%s&notes=%s&showDialog=true", cmdutil.ApplicationURL, ch.Config.Organization, url.QueryEscape(createReq.Name), url.QueryEscape(createReq.Notes)))
				if err != nil {
					return err
				}
				return nil
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := cmdutil.PrintProgress("Creating database...")
			defer end()
			database, err := client.Databases.Create(ctx, createReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("organization %s does not exist\n", cmdutil.BoldBlue(ch.Config.Organization))
				case planetscale.ErrResponseMalformed:
					return cmdutil.MalformedError(err)
				default:
					return err
				}
			}

			end()
			if ch.Config.OutputJSON {
				err := printer.PrintJSON(database)
				if err != nil {
					return err
				}
			} else {
				fmt.Printf("Database %s was successfully created!\n", cmdutil.BoldBlue(database.Name))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&createReq.Notes, "notes", "", "notes for the database")
	cmd.Flags().BoolP("web", "w", false, "Create a database in your web browser")

	return cmd
}
