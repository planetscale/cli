package printer

import ps "github.com/planetscale/planetscale-go"

type DatabaseBranchStatus struct {
	Status      string `header:"status" json:"status"`
	GatewayHost string `header:"gateway_host" json:"gateway_host"`
	GatewayPort int    `header:"gateway_port,text" json:"gateway_port"`
	User        string `header:"username" json:"user"`
	Password    string `header:"password" json:"password"`
}

func NewDatabaseBranchStatusPrinter(status *ps.DatabaseBranchStatus) *DatabaseBranchStatus {
	return &DatabaseBranchStatus{
		Status:      status.DeployPhase,
		GatewayHost: status.GatewayHost,
		GatewayPort: status.GatewayPort,
		User:        status.User,
		Password:    status.Password,
	}
}
