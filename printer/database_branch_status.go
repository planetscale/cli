package printer

import ps "github.com/planetscale/planetscale-go"

type DatabaseBranchStatus struct {
	DeployPhase string `header:"status"`
	GatewayHost string `header:"gateway_host"`
	GatewayPort int    `header:"gateway_port,text"`
	User        string `header:"username"`
	Password    string `header:"password"`
}

func NewDatabaseBranchStatusPrinter(status *ps.DatabaseBranchStatus) *DatabaseBranchStatus {
	return &DatabaseBranchStatus{
		DeployPhase: status.DeployPhase,
		GatewayHost: status.GatewayHost,
		GatewayPort: status.GatewayPort,
		User:        status.User,
		Password:    status.Password,
	}
}
