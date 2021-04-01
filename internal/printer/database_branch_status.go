package printer

import ps "github.com/planetscale/planetscale-go/planetscale"

type DatabaseBranchStatus struct {
	Status      string `header:"status" json:"status"`
	GatewayHost string `header:"gateway_host" json:"gateway_host"`
	GatewayPort int    `header:"gateway_port,text" json:"gateway_port"`
	User        string `header:"username" json:"user"`
	Password    string `header:"password" json:"password"`
}

func NewDatabaseBranchStatusPrinter(status *ps.DatabaseBranchStatus) *ObjectPrinter {
	return &ObjectPrinter{
		Source:  status,
		Printer: newDatabaseBranchStatusPrinter(status),
	}
}

func newDatabaseBranchStatusPrinter(status *ps.DatabaseBranchStatus) *DatabaseBranchStatus {
	var ready = "ready"
	if !status.Ready {
		ready = "not ready"
	}
	return &DatabaseBranchStatus{
		Status:      ready,
		GatewayHost: status.Credentials.GatewayHost,
		GatewayPort: status.Credentials.GatewayPort,
		User:        status.Credentials.User,
		Password:    status.Credentials.Password,
	}
}
