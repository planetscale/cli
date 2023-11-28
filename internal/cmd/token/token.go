package token

import (
	"encoding/json"
	"fmt"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// Service token endpoints cannot be accessed when authed with a service token.
// If a user tries, show them this error message.
func checkServiceToken(cmd *cobra.Command, args []string, ch *cmdutil.Helper) error {
	if ch.Config.ServiceTokenIsSet() {
		return fmt.Errorf("%s is unavailable when authenticated with a service token", printer.BoldBlue(cmd.CommandPath()))
	}
	return nil
}

// TokenCmd encapsulates the command for managing service tokens.
func TokenCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service-token <action>",
		Short: "Create, list, and manage access for service tokens",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.CheckAuthentication(ch.Config)(cmd, args); err != nil {
				return err
			}

			return checkServiceToken(cmd, args, ch)
		},
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(CreateCmd(ch))
	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(ShowAccessCmd(ch))
	cmd.AddCommand(AddAccessCmd(ch))
	cmd.AddCommand(DeleteAccessCmd(ch))
	cmd.AddCommand(DeleteCmd(ch))
	return cmd
}

// ServiceToken returns a table and json serializable schema snapshot.
type ServiceToken struct {
	ID    string `header:"id" json:"id"`
	Token string `header:"token" json:"token"`

	orig *ps.ServiceToken
}

func (s *ServiceToken) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(s.orig, "", "  ")
}

// toServiceToken returns a struct that prints out the various fields
// of a schema snapshot model.
func toServiceToken(st *ps.ServiceToken) *ServiceToken {
	return &ServiceToken{
		ID:    st.ID,
		Token: st.Token,
		orig:  st,
	}
}

func toServiceTokens(serviceTokens []*ps.ServiceToken) []*ServiceToken {
	snapshots := make([]*ServiceToken, 0, len(serviceTokens))

	for _, st := range serviceTokens {
		snapshots = append(snapshots, toServiceToken(st))
	}

	return snapshots
}

// ServiceTokenAccess returns a table and json serializiable service token
type ServiceTokenAccess struct {
	Database string   `header:"database" json:"database"`
	Accesses []string `header:"accesses" json:"accesses"`
}

func toServiceTokenAccesses(st []*ps.ServiceTokenAccess) []*ServiceTokenAccess {
	data := map[string]*ServiceTokenAccess{}
	for _, v := range st {
		if db, ok := data[v.Resource.Name]; ok {
			db.Accesses = append(db.Accesses, v.Access)
		} else {
			data[v.Resource.Name] = &ServiceTokenAccess{
				Database: v.Resource.Name,
				Accesses: []string{v.Access},
			}
		}
	}

	out := make([]*ServiceTokenAccess, 0, len(data))
	for _, v := range data {
		out = append(out, v)
	}
	return out
}

// ServiceTokenGrant erturns a table and json serializable service token grant
type ServiceTokenGrant struct {
	ResourceName string   `header:"resource_name" json:"resource_name"`
	ResourceType string   `header:"resource_type" json:"resource_type"`
	Accesses     []string `header:"accesses" json:"accesses"`
}

func toServiceTokenGrants(st []*ps.ServiceTokenGrant) []*ServiceTokenGrant {
	out := make([]*ServiceTokenGrant, 0, len(st))
	for _, v := range st {
		accesses := make([]string, 0, len(v.Accesses))
		for _, a := range v.Accesses {
			accesses = append(accesses, a.Access)
		}
		out = append(out, &ServiceTokenGrant{
			ResourceName: v.ResourceName,
			ResourceType: v.ResourceType,
			Accesses:     accesses,
		})
	}

	return out
}
