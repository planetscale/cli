package password

import (
	"errors"
	"fmt"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
)

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		role string
		ttl  time.Duration
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

			if flags.role != "" {
				_, err := cmdutil.RoleFromString(flags.role)
				if err != nil {
					return err
				}
			}

			ttl, err := ttlSeconds(flags.ttl)
			if err != nil {
				return err
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
				TTL:          ttl,
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
		"admin", "Role defines the access level, allowed values are : reader, writer, readwriter, admin. By default it is admin.")
	cmd.PersistentFlags().DurationVar(&flags.ttl, "ttl", 0*time.Second, "TTL defines the time to live for the password, rounded to the nearest second. By default it is 0 which means it will never expire.")
	return cmd
}

// ttlSeconds validates and converts a duration TTL into an integer TTL in
// seconds.
func ttlSeconds(ttl time.Duration) (int, error) {
	switch {
	case ttl < 0:
		return 0, errors.New("TTL cannot be negative")
	case ttl > 0 && ttl < 1*time.Second:
		return 0, errors.New("TTL must be at least 1 second")
	default:
		return int(ttl.Round(1 * time.Second).Seconds()), nil
	}
}
