package password

import (
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		role    string
		ttl     cmdutil.TTLFlag
		replica bool
	}

	cmd := &cobra.Command{
		Use:     "create <database> <branch> <name>",
		Short:   "Create password to access a branch's data",
		Args:    cmdutil.RequiredArgs("database", "branch", "name"),
		Aliases: []string{"p"},
		RunE: func(cmd *cobra.Command, args []string) error {
			database := args[0]
			branch := args[1]
			name := args[2]

			if flags.role == "" {
				if flags.replica {
					flags.role = "reader"
				} else {
					// Maintain old behavior - "admin" is the default role.
					flags.role = "admin"
				}
			}

			if flags.role != "" {
				_, err := cmdutil.RoleFromString(flags.role)
				if err != nil {
					return err
				}
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Creating password of %s/%s...", printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			pass, err := client.Passwords.Create(cmd.Context(), &ps.DatabaseBranchPasswordRequest{
				Database:     database,
				Branch:       branch,
				Organization: ch.Config.Organization,
				Name:         name,
				Role:         flags.role,
				TTL:          int(flags.ttl.Value.Seconds()),
				Replica:      flags.replica,
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
				saveWarning := printer.BoldRed("Please save the values below as they will not be shown again")
				ch.Printer.Printf("Password %s was successfully created in %s/%s.\n%s\n\n",
					printer.BoldBlue(pass.Name), printer.BoldBlue(database), printer.BoldBlue(branch), saveWarning)
			}

			return ch.Printer.PrintResource(toPasswordWithPlainText(pass))
		},
	}
	cmd.PersistentFlags().StringVar(&flags.role, "role",
		"", "Role defines the access level, allowed values are: reader, writer, readwriter, admin. Defaults to 'reader' for replica passwords, otherwise defaults to 'admin'.")
	cmd.PersistentFlags().Var(&flags.ttl, "ttl", `TTL defines the time to live for the password. Durations such as "30m", "24h", or bare integers such as "3600" (seconds) are accepted. The default TTL is 0s, which means the password will never expire.`)
	cmd.Flags().BoolVar(&flags.replica, "replica", false, "When enabled, the password will route all reads to the branch's primary replicas and all read-only regions.")

	return cmd
}
