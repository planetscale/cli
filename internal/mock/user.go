package mock

import (
	"context"

	ps "github.com/planetscale/cli/internal/planetscale"
)

type UsersService struct {
	GetCurrentFn        func(context.Context) (*ps.User, error)
	GetCurrentFnInvoked bool
}

func (u *UsersService) GetCurrent(ctx context.Context) (*ps.User, error) {
	u.GetCurrentFnInvoked = true
	return u.GetCurrentFn(ctx)
}
