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
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(CreateCmd(ch))
	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(DeleteCmd(ch))

	return cmd
}

type Passwords []*Password

type Password struct {
	PublicID  string `header:"id" json:"id"`
	Name      string `header:"name" json:"name"`
	Branch    string `header:"branch" json:"branch"`
	Username  string `header:"username" json:"username"`
	Role      string `header:"role" json:"role"`
	RoleDesc  string `header:"role description" json:"-"`
	TTL       int    `header:"ttl" json:"ttl"`
	CreatedAt int64  `json:"created_at"`
	orig      *ps.DatabaseBranchPassword
}

type PasswordWithPlainText struct {
	Name              string               `header:"name" json:"name"`
	Branch            string               `header:"branch" json:"branch"`
	PublicID          string               `header:"id" json:"public_id"`
	Username          string               `header:"username" json:"username"`
	AccessHostUrl     string               `header:"access host url" json:"access_host_url"`
	Role              string               `header:"role" json:"role"`
	RoleDesc          string               `header:"role description" json:"role_description"`
	PlainText         string               `header:"password" json:"password"`
	TTL               int                  `header:"ttl" json:"ttl"`
	ConnectionStrings ps.ConnectionStrings `json:"connection_strings"`
	orig              *ps.DatabaseBranchPassword
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
	return &Password{
		Name:      password.Name,
		Branch:    password.Branch.Name,
		PublicID:  password.PublicID,
		Username:  password.Username,
		Role:      password.Role,
		RoleDesc:  toRoleDesc(password.Role),
		TTL:       password.TTL,
		CreatedAt: password.CreatedAt.UTC().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond)),
		orig:      password,
	}
}

func toPasswords(passwords []*ps.DatabaseBranchPassword) []*Password {
	bs := make([]*Password, 0, len(passwords))
	for _, password := range passwords {
		bs = append(bs, toPassword(password))
	}
	return bs
}

func toPasswordWithPlainText(password *ps.DatabaseBranchPassword) *PasswordWithPlainText {
	return &PasswordWithPlainText{
		Name:              password.Name,
		Branch:            password.Branch.Name,
		PublicID:          password.PublicID,
		Username:          password.Username,
		PlainText:         password.PlainText,
		AccessHostUrl:     password.Hostname,
		Role:              password.Role,
		RoleDesc:          toRoleDesc(password.Role),
		TTL:               password.TTL,
		ConnectionStrings: password.ConnectionStrings,
		orig:              password,
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
