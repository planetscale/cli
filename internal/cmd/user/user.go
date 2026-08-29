package user

import (
	"encoding/json"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func UserCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "user <command>",
		Short:             "Show the currently authenticated user",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.AddCommand(ShowCmd(ch))

	return cmd
}

type User struct {
	ID                      string `header:"id" json:"id"`
	DisplayName             string `header:"display_name" json:"display_name"`
	Name                    string `header:"name" json:"name"`
	Email                   string `header:"email" json:"email"`
	TwoFactorAuthConfigured bool   `header:"two_factor_auth_configured" json:"two_factor_auth_configured"`
	CreatedAt               int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt               int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.User
}

func (u *User) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(u.orig, "", "  ")
}

func toUser(user *ps.User) *User {
	return &User{
		ID:                      user.ID,
		DisplayName:             user.DisplayName,
		Name:                    user.Name,
		Email:                   user.Email,
		TwoFactorAuthConfigured: user.TwoFactorAuthConfigured,
		CreatedAt:               printer.GetMilliseconds(user.CreatedAt),
		UpdatedAt:               printer.GetMilliseconds(user.UpdatedAt),
		orig:                    user,
	}
}
