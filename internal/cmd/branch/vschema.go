package branch

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// VSchemaCmd is the command for showing the schema of a branch.
func VSchemaCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		keyspace string
	}

	cmd := &cobra.Command{
		Use:   "vschema <database> <branch>",
		Short: "Show the vschema of a branch",
		Args:  cmdutil.RequiredArgs("database", "branch"),
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
					return fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
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
	cmd.Flags().MarkHidden("keyspace")

	return cmd
}
