package planetscale

import (
	"context"
	"fmt"
	"net/http"
)

const usersAPIPath = "v1/user"

// User is the authenticated PlanetScale user.
type User struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Name        string `json:"name"`
	Email       string `json:"email"`
}

// UsersService is an interface for communicating with the PlanetScale
// Users API endpoints.
type UsersService interface {
	GetCurrent(context.Context) (*User, error)
}

type usersService struct {
	client *Client
}

var _ UsersService = &usersService{}

func (u *usersService) GetCurrent(ctx context.Context) (*User, error) {
	req, err := u.client.newRequest(http.MethodGet, usersAPIPath, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for get current user: %w", err)
	}

	user := &User{}
	if err := u.client.do(ctx, req, user); err != nil {
		return nil, err
	}

	return user, nil
}
