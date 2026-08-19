package password

import (
	"errors"
	"fmt"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"

	ps "github.com/planetscale/cli/internal/planetscale"

	"github.com/spf13/cobra"
)

// UpdateCmd updates the name or IP restrictions of a branch password. It never
// changes the password secret; use create and delete, or renew, for that.
func UpdateCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		name    string
		newName string
		cidrs   []string
	}

	cmd := &cobra.Command{
		Use:   "update <database> <branch> [<password-id>]",
		Short: "Update a branch password's name or IP restrictions",
		Long: `Update a branch password's name or IP restrictions.

This does not change the password itself. To rotate a password, create a new
one and delete the old one.`,
		Example: `  pscale password update mydb main pscale_pw_xxx --new-name reporting
  pscale password update mydb main --name reporting --cidrs 10.0.0.0/8,192.168.1.1/32
  pscale password update mydb main --name reporting --cidrs ""`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			database := args[0]
			branch := args[1]

			if err := passwordSelector(args, flags.name); err != nil {
				return err
			}

			newNameSet := cmd.Flags().Changed("new-name")
			cidrsSet := cmd.Flags().Changed("cidrs")
			if !newNameSet && !cidrsSet {
				return errors.New("at least one of --new-name or --cidrs must be provided")
			}
			if newNameSet && flags.newName == "" {
				return errors.New("--new-name cannot be empty")
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			var argID string
			if len(args) == 3 {
				argID = args[2]
			}

			passwordId, err := resolvePasswordID(ctx, ch, client, database, branch, argID, flags.name)
			if err != nil {
				return err
			}

			updateReq := &ps.UpdateDatabaseBranchPasswordRequest{
				Organization: ch.Config.Organization,
				Database:     database,
				Branch:       branch,
				PasswordId:   passwordId,
			}
			if newNameSet {
				updateReq.Name = flags.newName
			}
			if cidrsSet {
				cidrs := normalizeCIDRs(flags.cidrs)
				updateReq.CIDRs = &cidrs
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Updating password %s in %s/%s",
				printer.BoldBlue(passwordId), printer.BoldBlue(database), printer.BoldBlue(branch)))
			defer end()

			password, err := client.Passwords.Update(ctx, updateReq)
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("password %s does not exist in branch %s of %s (organization: %s)",
						printer.BoldBlue(passwordId), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}

			end()

			if ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("Password %s was successfully updated in %s.\n",
					printer.BoldBlue(password.Name), printer.BoldBlue(branch))
				if cidrsSet {
					if len(password.CIDRs) == 0 {
						ch.Printer.Println("IP restrictions: none")
					} else {
						ch.Printer.Printf("IP restrictions: %s\n", strings.Join(password.CIDRs, ", "))
					}
				}
				return nil
			}

			return ch.Printer.PrintResource(toPassword(password))
		},
	}

	cmd.Flags().StringVar(&flags.name, "name", "", "Update password by name instead of ID")
	cmd.Flags().StringVar(&flags.newName, "new-name", "", "New name for the password")
	cmd.Flags().StringSliceVar(&flags.cidrs, "cidrs", nil,
		"Replace the IP addresses and CIDR ranges allowed to use this password. Pass an empty value to remove all restrictions")

	return cmd
}

// normalizeCIDRs drops the empty entries produced by flags like --cidrs "" so
// that clearing restrictions sends an empty list instead of a blank CIDR.
func normalizeCIDRs(cidrs []string) []string {
	out := make([]string, 0, len(cidrs))
	for _, cidr := range cidrs {
		if trimmed := strings.TrimSpace(cidr); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
