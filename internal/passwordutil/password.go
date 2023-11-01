package passwordutil

import (
	"context"
	"fmt"
	"time"

	nanoid "github.com/matoous/go-nanoid/v2"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

type Options struct {
	Organization string
	Database     string
	Branch       string
	Role         cmdutil.PasswordRole
	Name         string
	TTL          time.Duration
}

type Password struct {
	ps.DatabaseBranchPassword
	cleanup func(context.Context) error
}

func (p *Password) Cleanup(ctx context.Context) error {
	if p.cleanup == nil {
		return nil
	}
	return p.cleanup(ctx)
}

func New(ctx context.Context, client *ps.Client, opt Options) (*Password, error) {
	pw, err := client.Passwords.Create(ctx, &ps.DatabaseBranchPasswordRequest{
		Organization: opt.Organization,
		Database:     opt.Database,
		Branch:       opt.Branch,
		Role:         opt.Role.ToString(),
		Name:         opt.Name,
		TTL:          int(opt.TTL.Seconds()),
	})
	if err != nil {
		return nil, err
	}

	return &Password{
		DatabaseBranchPassword: *pw,
		cleanup: func(ctx context.Context) error {
			return client.Passwords.Delete(ctx, &ps.DeleteDatabaseBranchPasswordRequest{
				Organization: opt.Organization,
				Database:     opt.Database,
				Branch:       opt.Branch,
				Name:         pw.Name,
				PasswordId:   pw.PublicID,
			})
		},
	}, nil
}

func GenerateName(prefix string) string {
	return fmt.Sprintf(
		"%s-%s-%s",
		prefix,
		time.Now().Format("2006-01-02"),
		nanoid.MustGenerate(publicIdAlphabet, publicIdLength),
	)
}

const (
	publicIdAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	publicIdLength   = 6
)
