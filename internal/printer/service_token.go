package printer

import (
	"github.com/planetscale/planetscale-go/planetscale"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

// ServiceToken returns a table and json serializable schema snapshot.
type ServiceToken struct {
	Name  string `header:"name" json:"name"`
	Token string `header:"token" json:"token"`
}

// NewServiceTokenPrinter returns a struct that prints out the various fields
// of a schema snapshot model.
func NewServiceTokenPrinter(st *planetscale.ServiceToken) *ServiceToken {
	return &ServiceToken{
		Name:  st.ID,
		Token: st.Token,
	}
}

func NewServiceTokenSlicePrinter(serviceTokens []*ps.ServiceToken) []*ServiceToken {
	snapshots := make([]*ServiceToken, 0, len(serviceTokens))

	for _, st := range serviceTokens {
		snapshots = append(snapshots, NewServiceTokenPrinter(st))
	}

	return snapshots
}
