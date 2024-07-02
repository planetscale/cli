package keyspace

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func VSchemaCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vschema <command>",
		Short: "Update or show the VSchema of a branch",
	}

	cmd.AddCommand(ShowVSchemaCmd(ch))
	cmd.AddCommand(UpdateVSchemaCmd(ch))

	return cmd
}

// ShowVSchemaCmd is the command for showing a keyspace's VSchema.
func ShowVSchemaCmd(ch *cmdutil.Helper) *cobra.Command {
	return &cobra.Command{
		Use:   "show <database> <branch> <keyspace>",
		Short: "Show the VSchema of a keyspace in a branch",
		Args:  cmdutil.RequiredArgs("database", "branch", "keyspace"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, keyspace := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			vSchema, err := client.Keyspaces.VSchema(ctx, &planetscale.GetKeyspaceVSchemaRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     keyspace,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("received HTTP 404 for branch %s in database %s (organization: %s). This may mean you're requesting a keyspace that does not exist",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			if ch.Printer.Format() != printer.Human {
				return ch.Printer.PrintResource(vSchema)
			}

			err = ch.Printer.PrettyPrintJSON([]byte(vSchema.Raw))
			if err != nil {
				return fmt.Errorf("reading vschema raw: %s", err)
			}

			return nil
		},
	}
}

// UpdateVSchemaCmd is the command for updating a keyspace's VSchema.
func UpdateVSchemaCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		vschema string
	}
	cmd := &cobra.Command{
		Use:   "update <database> <branch> <keyspace> --vschema <file>",
		Short: "Update the VSchema of a keyspace",
		Args:  cmdutil.RequiredArgs("database", "branch", "keyspace"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch, keyspace := args[0], args[1], args[2]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			var data string

			if flags.vschema != "" {
				rawVSchema, err := os.ReadFile(flags.vschema)
				if err != nil {
					return err
				}
				data = string(rawVSchema)
			} else {
				stdinFile, err := os.Stdin.Stat()
				if err != nil {
					return err
				}

				if (stdinFile.Mode() & os.ModeCharDevice) == 0 {
					stdin, err := io.ReadAll(os.Stdin)
					if err != nil {
						return err
					}

					data = string(stdin)
				}
			}

			if len(data) == 0 {
				return errors.New("no vschema provided, use the --vschema and provide a file or pipe the vschema to standard in")
			}

			vschema, err := client.Keyspaces.UpdateVSchema(ctx, &planetscale.UpdateKeyspaceVSchemaRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     keyspace,
				VSchema:      data,
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case planetscale.ErrNotFound:
					return fmt.Errorf("received HTTP 404 for branch %s in database %s (org: %s). This may mean you're requesting a keyspace that does not exist or not supplying one if you have multiple",
						printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			if ch.Printer.Format() != printer.Human {
				return ch.Printer.PrintResource(vschema)
			}
			err = ch.Printer.PrettyPrintJSON([]byte(vschema.Raw))
			if err != nil {
				return fmt.Errorf("reading vschema raw: %s", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.vschema, "vschema", "", "The path to the VSchema file")

	return cmd
}
