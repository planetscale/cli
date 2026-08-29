package planetscale

import (
	"context"
	"net/http"
	"time"
)

const userAPIPath = "v1/user"

// UsersService is an interface for communicating with the PlanetScale Users API.
type UsersService interface {
	GetCurrentUser(context.Context) (*User, error)
}

// User represents a PlanetScale user.
type User struct {
	ID                      string        `json:"id"`
	DisplayName             string        `json:"display_name"`
	Name                    string        `json:"name"`
	Email                   string        `json:"email"`
	AvatarURL               string        `json:"avatar_url"`
	TwoFactorAuthConfigured bool          `json:"two_factor_auth_configured"`
	CreatedAt               time.Time     `json:"created_at"`
	UpdatedAt               time.Time     `json:"updated_at"`
	DefaultOrganization     *Organization `json:"default_organization,omitempty"`
}

type usersService struct {
	client *Client
}

var _ UsersService = &usersService{}

func (u *usersService) GetCurrentUser(ctx context.Context) (*User, error) {
	req, err := u.client.newRequest(http.MethodGet, userAPIPath, nil)
	if err != nil {
		return nil, err
	}

	user := &User{}
	if err := u.client.do(ctx, req, &user); err != nil {
		return nil, err
	}

	return user, nil
}
