package printer

import (
	"github.com/planetscale/planetscale-go/planetscale"
)

// ServiceTokenAccess returns a table and json serializable schema snapshot.
type ServiceTokenAccess struct {
	Database string   `header:"database" json:"database"`
	Accesses []string `header:"accesses" json:"accesses"`
}

// NewServiceTokenPrinter returns a struct that prints out the various fields
// of a schema snapshot model.
func NewServiceTokenAccessPrinter(st []*planetscale.ServiceTokenAccess) []*ServiceTokenAccess {
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
