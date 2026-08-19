package password

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lensesio/tableprinter"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"

	ps "github.com/planetscale/cli/internal/planetscale"
)

// PasswordCmd handles branch passwords.
func PasswordCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "password <command>",
		Short:             "Create, list, update, and delete branch passwords",
		Long:              "Create, list, update, and delete branch passwords.\n\nThis command is only supported for Vitess databases.",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(CreateCmd(ch))
	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(ShowCmd(ch))
	cmd.AddCommand(UpdateCmd(ch))
	cmd.AddCommand(DeleteCmd(ch))
	cmd.AddCommand(RenewCmd(ch))

	return cmd
}

// resolvePasswordID resolves the password ID for commands that accept either a
// password ID argument or a --name flag. Exactly one of id or name must be set.
func resolvePasswordID(ctx context.Context, ch *cmdutil.Helper, client *ps.Client, database, branch, id, name string) (string, error) {
	if name == "" {
		return id, nil
	}

	end := ch.Printer.PrintProgress(fmt.Sprintf("Finding password %s in %s/%s",
		printer.BoldBlue(name), printer.BoldBlue(database), printer.BoldBlue(branch)))
	defer end()

	perPage := 100
	for page := 1; ; page++ {
		passwords, err := client.Passwords.List(ctx, &ps.ListDatabaseBranchPasswordRequest{
			Organization: ch.Config.Organization,
			Database:     database,
			Branch:       branch,
		}, ps.WithPage(page), ps.WithPerPage(perPage))
		if err != nil {
			switch cmdutil.ErrCode(err) {
			case ps.ErrNotFound:
				return "", fmt.Errorf("branch %s does not exist in database %s (organization: %s)",
					printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
			default:
				return "", cmdutil.HandleError(err)
			}
		}

		for _, password := range passwords {
			if password.Name == name {
				return password.PublicID, nil
			}
		}

		if len(passwords) < perPage {
			return "", fmt.Errorf("password with name %s does not exist in branch %s of %s (organization: %s)",
				printer.BoldBlue(name), printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
		}
	}
}

// passwordSelector validates the password-id argument and --name flag pair used
// by show, update, and delete.
func passwordSelector(args []string, name string) error {
	if name != "" && len(args) == 3 {
		return errors.New("cannot specify both password-id argument and --name flag")
	}
	if name == "" && len(args) != 3 {
		return errors.New("must provide either password-id argument or --name flag")
	}
	return nil
}

type Passwords []*Password

type Password struct {
	PublicID       string `header:"id" json:"id"`
	Name           string `header:"name" json:"name"`
	Branch         string `header:"branch" json:"branch"`
	Username       string `header:"username" json:"username"`
	Role           string `header:"role" json:"role"`
	RoleDesc       string `header:"role description" json:"-"`
	ConnectionType string `header:"connection type" json:"connection_type"`
	TTL            int    `header:"ttl" json:"ttl"`
	Remaining      int    `header:"ttl_remaining" json:"-"`
	CreatedAt      int64  `json:"created_at"`
	Expired        bool   `header:"expired" json:"expired"`
	orig           *ps.DatabaseBranchPassword
}

type passwordWithoutTTL struct {
	PublicID       string `header:"id" json:"id"`
	Name           string `header:"name" json:"name"`
	Branch         string `header:"branch" json:"branch"`
	Username       string `header:"username" json:"username"`
	Role           string `header:"role" json:"role"`
	RoleDesc       string `header:"role description" json:"-"`
	ConnectionType string `header:"connection type" json:"connection_type"`
	CreatedAt      int64  `json:"created_at"`
	orig           *ps.DatabaseBranchPassword
}

type PasswordWithPlainText struct {
	Name           string `header:"name" json:"name"`
	Branch         string `header:"branch" json:"branch"`
	PublicID       string `header:"id" json:"public_id"`
	Username       string `header:"username" json:"username"`
	AccessHostUrl  string `header:"access host url" json:"access_host_url"`
	Role           string `header:"role" json:"role"`
	RoleDesc       string `header:"role description" json:"role_description"`
	ConnectionType string `header:"connection type" json:"connection_type"`
	PlainText      string `header:"password" json:"password"`
	TTL            int    `header:"ttl" json:"ttl"`
	orig           *ps.DatabaseBranchPassword
}

func (b *Password) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(b.orig, "", "  ")
}

func (b *Password) MarshalCSVValue() interface{} {
	return []*Password{b}
}

func (b Passwords) String() string {
	var buf strings.Builder
	tableprinter.Print(&buf, b)
	return buf.String()
}

func (b *PasswordWithPlainText) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(b.orig, "", "  ")
}

func (b *PasswordWithPlainText) MarshalCSVValue() interface{} {
	return []*PasswordWithPlainText{b}
}

// toPassword Returns a struct that prints out the various fields of a branch model.
func toPassword(password *ps.DatabaseBranchPassword) *Password {
	ttlRemaining := 0
	if password.TTL > 0 {
		ttlRemaining = max(int(time.Until(password.ExpiresAt).Seconds()), 0)
	}
	return &Password{
		Name:           password.Name,
		Branch:         password.Branch.Name,
		PublicID:       password.PublicID,
		Username:       password.Username,
		Role:           password.Role,
		RoleDesc:       toRoleDesc(password.Role),
		ConnectionType: toConnectionTypeDesc(password),
		TTL:            password.TTL,
		Remaining:      ttlRemaining,
		CreatedAt:      toTimestamp(password.CreatedAt),
		Expired:        password.TTL > 0 && ttlRemaining == 0,
		orig:           password,
	}
}

func toPasswordWithoutTTL(password *ps.DatabaseBranchPassword) *passwordWithoutTTL {
	return &passwordWithoutTTL{
		Name:           password.Name,
		Branch:         password.Branch.Name,
		PublicID:       password.PublicID,
		Username:       password.Username,
		Role:           password.Role,
		RoleDesc:       toRoleDesc(password.Role),
		ConnectionType: toConnectionTypeDesc(password),
		CreatedAt:      toTimestamp(password.CreatedAt),
		orig:           password,
	}
}

// hasEphemeral checks if any password is emphemeral or not. Ephemeral is
// any password that has a TTL > 0. A 0 TTL password doesn't expire.
func hasEphemeral(passwords []*ps.DatabaseBranchPassword) bool {
	for _, password := range passwords {
		if password.TTL > 0 {
			return true
		}
	}
	return false
}

func toPasswords(passwords []*ps.DatabaseBranchPassword) []*Password {
	bs := make([]*Password, 0, len(passwords))
	for _, password := range passwords {
		bs = append(bs, toPassword(password))
	}
	return bs
}

func toPasswordsWithoutTTL(passwords []*ps.DatabaseBranchPassword) []*passwordWithoutTTL {
	bs := make([]*passwordWithoutTTL, 0, len(passwords))
	for _, password := range passwords {
		bs = append(bs, toPasswordWithoutTTL(password))
	}
	return bs
}

func toPasswordWithPlainText(password *ps.DatabaseBranchPassword) *PasswordWithPlainText {
	return &PasswordWithPlainText{
		Name:           password.Name,
		Branch:         password.Branch.Name,
		PublicID:       password.PublicID,
		Username:       password.Username,
		PlainText:      password.PlainText,
		AccessHostUrl:  password.Hostname,
		Role:           password.Role,
		RoleDesc:       toRoleDesc(password.Role),
		ConnectionType: toConnectionTypeDesc(password),
		TTL:            password.TTL,
		orig:           password,
	}
}

func toRoleDesc(role string) string {
	switch role {
	case "reader":
		return "Can Read"
	case "writer":
		return "Can Write"
	case "readwriter":
		return "Can Read & Write"
	case "admin":
		return "Can Read, Write & Administer"
	}
	return "Can Read"
}

func toConnectionTypeDesc(password *ps.DatabaseBranchPassword) string {
	if password.Replica {
		return "Replica"
	}
	if password.ReadOnlyRegion {
		if password.Region.Slug != "" {
			return "Read-only region (" + password.Region.Slug + ")"
		}
		return "Read-only region"
	}
	return "Primary"
}

func toTimestamp(t time.Time) int64 {
	return t.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}
