package token

import (
	"encoding/json"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// TokenCmd encapsulates the command for running snapshots.
func TokenCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service-token <action>",
		Short: "Create, get, and list service tokens",
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(CreateCmd(ch))
	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(ShowCmd(ch))
	cmd.AddCommand(AddAccessCmd(ch))
	cmd.AddCommand(DeleteAccessCmd(ch))
	cmd.AddCommand(DeleteCmd(ch))
	return cmd
}

// ServiceToken returns a table and json serializable schema snapshot.
type ServiceToken struct {
	Name  string `header:"name" json:"name"`
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
		Name:  st.ID,
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

// ServiceTokenAccess returns a table and json serializable schema snapshot.
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
