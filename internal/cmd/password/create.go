package password

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func CreateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		role    string
		ttl     ttlFlag
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
		"admin", "Role defines the access level, allowed values are : reader, writer, readwriter, admin. By default it is admin.")
	cmd.PersistentFlags().Var(&flags.ttl, "ttl", `TTL defines the time to live for the password. Durations such as "30m", "24h", or bare integers such as "3600" (seconds) are accepted. The default TTL is 0s, which means the password will never expire.`)
	cmd.Flags().BoolVar(&flags.replica, "replica", false, "When enabled, the password will route all reads to the branch's primary replicas and all read-only regions.")
	cmd.Flags().MarkHidden("replica")

	return cmd
}

var _ pflag.Value = &ttlFlag{}

// A ttlFlag is a pflag.Value specialized for parsing TTL durations which may or
// may not have an accompanying time unit.
type ttlFlag struct{ Value time.Duration }

func (f *ttlFlag) String() string { return f.Value.String() }
func (f *ttlFlag) Type() string   { return "duration" }

func (f *ttlFlag) Set(value string) error {
	if value == "" {
		// Empty string or undefined.
		return f.set(0 * time.Second)
	}

	if d, err := parseDuration(value); err == nil {
		// Valid stdlib duration.
		return f.set(d)
	}

	// Fall back to parsing a bare integer in seconds for CLI compatibility.
	i, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("cannot parse %q as TTL in seconds", value)
	}

	return f.set(time.Duration(i) * time.Second)
}

// set sets d into f after performing validation.
func (f *ttlFlag) set(d time.Duration) error {
	switch {
	case d < 0:
		return errors.New("TTL cannot be negative")
	case d.Round(time.Second) != d:
		return errors.New("TTL must be defined in increments of 1 second")
	default:
		f.Value = d
		return nil
	}
}

// parseDuration extends time.ParseDuration with a "d" unit that is not
// permitted by the Go standard library. For more information, see:
// https://github.com/golang/go/issues/11473.
func parseDuration(s string) (time.Duration, error) {
	// This is a very rudimentary parser; just look for a single rune suffix and
	// match on that using the generally accepted definitions of "day" with no
	// accounting for leap seconds.
	//
	// If these edge cases matter, users can always perform the math and enter
	// an absolute TTL in seconds.
	var multiplier time.Duration
	switch s[len(s)-1] {
	case 'd':
		multiplier = 24 * time.Hour
	default:
		return time.ParseDuration(s)
	}

	v, err := strconv.Atoi(s[:len(s)-1])
	if err != nil {
		return 0, err
	}

	return time.Duration(v) * multiplier, nil
}
