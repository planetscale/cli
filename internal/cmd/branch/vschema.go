package branch

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

// VSchemaCmd is the top-level command for fetching or updating the VSchema of a branch.
func VSchemaCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:        "vschema <command>",
		Deprecated: "use `pscale keyspace vschema` instead",
		Short:      "Fetch or update your keyspace VSchema",
		RunE: func(cmd *cobra.Command, args []string) error {
			// keep this around for backwards compat.
			return GetVSchemaCmd(ch).RunE(cmd, args)
		},
	}

	cmd.AddCommand(GetVSchemaCmd(ch))
	cmd.AddCommand(UpdateVSchemaCmd(ch))

	return cmd
}

// GetVSchemaCmd is the command for showing the VSchema of a branch.
func GetVSchemaCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		keyspace string
	}

	cmd := &cobra.Command{
		Use:        "show <database> <branch>",
		Short:      "Show the vschema of a branch",
		Deprecated: "use `pscale keyspace vschema show` instead",
		Args:       cmdutil.RequiredArgs("database", "branch"),
		Aliases:    []string{"get"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

			client, err := ch.Client()
			if err != nil {
				return err
			}

			vschema, err := client.DatabaseBranches.VSchema(ctx, &planetscale.BranchVSchemaRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     flags.keyspace,
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
				return ch.Printer.PrintResource(vschema)
			}

			err = ch.Printer.PrettyPrintJSON([]byte(vschema.Raw))
			if err != nil {
				return fmt.Errorf("reading vschema raw: %s", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "The keyspace in the branch")

	return cmd
}

// UpdateVSchemaCmd is the command for updating the VSchema of a branch.
func UpdateVSchemaCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		keyspace string
		vschema  string
	}

	cmd := &cobra.Command{
		Use:        "update <database> <branch> --vschema <file>",
		Short:      "Update the vschema of a branch",
		Deprecated: "use `pscale keyspace vschema update` instead",
		Args:       cmdutil.RequiredArgs("database", "branch"),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database, branch := args[0], args[1]

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

			vschema, err := client.DatabaseBranches.UpdateVSchema(ctx, &planetscale.UpdateBranchVschemaRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				Keyspace:     flags.keyspace,
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

	cmd.Flags().StringVar(&flags.vschema, "vschema", "", "The vschema to set in JSON format")
	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "The keyspace to apply the vschema to")

	return cmd
}
