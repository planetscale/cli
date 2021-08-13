package passwords

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
	UserName  string `header:"username" json:"username"`
	Role      string `header:"role" json:"role"`
	RoleDesc  string `header:"role description" json:"-"`
	CreatedAt int64  `json:"created_at"`
	orig      *ps.DatabaseBranchPassword
}

type DeletedPassword struct {
	Name      string `header:"name" json:"name"`
	Role      string `header:"role" json:"role"`
	CreatedAt int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	DeletedAt int64  `header:"deleted_at,timestamp(ms|utc|human)" json:"deleted_at"`
	orig      *ps.DatabaseBranchPassword
}

type PasswordWithPlainText struct {
	Name              string               `header:"name" json:"name"`
	PublicID          string               `header:"username" json:"username"`
	AccessHostUrl     string               `header:"access host url" json:"access_host_url"`
	Role              string               `header:"role" json:"role"`
	RoleDesc          string               `header:"role description" json:"role_description"`
	PlainText         string               `header:"plain text" json:"plain_text"`
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

// toPassword Returns a struct that prints out the various fields of a branch model.
func toPassword(password *ps.DatabaseBranchPassword) *Password {
	return &Password{
		Name:      password.Name,
		PublicID:  password.PublicID,
		UserName:  password.PublicID,
		Role:      password.Role,
		RoleDesc:  toRoleDesc(password.Role),
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
		PublicID:          password.PublicID,
		PlainText:         password.PlainText,
		AccessHostUrl:     password.Branch.AccessHostURL,
		Role:              password.Role,
		RoleDesc:          toRoleDesc(password.Role),
		ConnectionStrings: password.ConnectionStrings,
		orig:              password,
	}
}

func toRoleDesc(role string) string {
	switch role {
	case "writer":
		return "Can Read & Write"
	case "admin":
		return "Can Read, Write & Administer"
	}
	return "no idea"
}
