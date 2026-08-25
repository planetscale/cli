package planetscale

import (
	"context"
	"fmt"
	"net/http"
)

const regionsAPIPath = "v1/regions"

type Region struct {
	ID                  string   `json:"id"`
	Slug                string   `json:"slug"`
	Provider            string   `json:"provider"`
	Name                string   `json:"display_name"`
	Location            string   `json:"location"`
	Enabled             bool     `json:"enabled"`
	IsDefault           bool     `json:"current_default"`
	PublicIPAddresses   []string `json:"public_ip_addresses"`
	MySQLSupported      bool     `json:"mysql_supported"`
	PostgreSQLSupported bool     `json:"postgresql_supported"`
}

type regionsResponse struct {
	Regions []*Region `json:"data"`
}

type ListRegionsRequest struct{}

type RegionsService interface {
	List(ctx context.Context, req *ListRegionsRequest) ([]*Region, error)
}

type regionsService struct {
	client *Client
}

var _ RegionsService = &regionsService{}

func NewRegionsSevice(client *Client) *regionsService {
	return &regionsService{
		client: client,
	}
}

func (r *regionsService) List(ctx context.Context, listReq *ListRegionsRequest) ([]*Region, error) {
	req, err := r.client.newRequest(http.MethodGet, regionsAPIPath, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for list regions: %w", err)
	}

	regionsResponse := &regionsResponse{}
	if err := r.client.do(ctx, req, &regionsResponse); err != nil {
		return nil, err
	}

	return regionsResponse.Regions, nil
}
