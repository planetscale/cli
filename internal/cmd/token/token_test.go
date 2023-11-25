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
