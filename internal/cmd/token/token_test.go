package token

import (
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	"github.com/spf13/cobra"
)

func TestServiceToken_TokenCmd_ServiceTokenAuth(t *testing.T) {
	c := qt.New(t)

	ch := &cmdutil.Helper{
		Config: &config.Config{
			ServiceTokenID: "token-id",
			ServiceToken:   "token",
		},
	}

	cmd := TokenCmd(ch)
	userCommand := &cobra.Command{}

	err := cmd.PersistentPreRunE(userCommand, []string{})

	c.Assert(err, qt.IsNotNil)
	c.Assert(err.Error(), qt.Equals, " is unavailable when authenticated with a service token")
}

func TestServiceToken_TokenCmdRegistersShowSeparatelyFromShowAccess(t *testing.T) {
	c := qt.New(t)

	cmd := TokenCmd(&cmdutil.Helper{Config: &config.Config{}})

	show, _, err := cmd.Find([]string{"show"})
	c.Assert(err, qt.IsNil)
	c.Assert(show.Name(), qt.Equals, "show")

	showAccess, _, err := cmd.Find([]string{"show-access"})
	c.Assert(err, qt.IsNil)
	c.Assert(showAccess.Name(), qt.Equals, "show-access")
}
