package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type UsersService struct {
	GetCurrentUserFn        func(context.Context) (*ps.User, error)
	GetCurrentUserFnInvoked bool
}

func (u *UsersService) GetCurrentUser(ctx context.Context) (*ps.User, error) {
	u.GetCurrentUserFnInvoked = true
	return u.GetCurrentUserFn(ctx)
}
