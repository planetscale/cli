package password

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/lensesio/tableprinter"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"

	ps "github.com/planetscale/planetscale-go/planetscale"
)

// PasswordCmd handles branch passwords.
func PasswordCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "password <command>",
		Short:             "Create, list, and delete branch passwords",
		Long:              "Create, list, and delete branch passwords.\n\nThis command is only supported for Vitess databases.",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(CreateCmd(ch))
	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(DeleteCmd(ch))
	cmd.AddCommand(RenewCmd(ch))

	return cmd
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
		ConnectionType: toConnectionTypeDesc(password.Replica),
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
		ConnectionType: toConnectionTypeDesc(password.Replica),
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
		ConnectionType: toConnectionTypeDesc(password.Replica),
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

func toConnectionTypeDesc(replica bool) string {
	if replica {
		return "Replica"
	} else {
		return "Primary"
	}
}

func toTimestamp(t time.Time) int64 {
	return t.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}
