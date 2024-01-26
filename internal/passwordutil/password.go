package passwordutil

import (
	"context"
	"fmt"
	"time"

	nanoid "github.com/matoous/go-nanoid/v2"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

const (
	publicIdAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	publicIdLength   = 6
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
	Password *ps.DatabaseBranchPassword
	cleanup  func(context.Context) error
	renew    func(context.Context) error
}

func (p *Password) Cleanup(ctx context.Context) error {
	if p.cleanup == nil {
		return nil
	}
	return p.cleanup(ctx)
}

func (p *Password) Renew(ctx context.Context) error {
	if p.renew == nil || p.Password.TTL == 0 {
		return nil
	}

	ttl := time.Duration(p.Password.TTL) * time.Second

	// renew at half the TTL
	timer := time.NewTimer(ttl / 2)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		}

		if err := p.renew(ctx); err != nil {
			switch cmdutil.ErrCode(err) {
			case ps.ErrNotFound, ps.ErrRetry:
				// either of these indicate there's no ability to retry
				return fmt.Errorf("password failed to renew: %w", err)
			}
			// on failure to renew, retry a bit more aggressively
			timer.Reset(ttl / 8)
		} else {
			timer.Reset(ttl / 2)
		}
	}
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
		Password: pw,
		cleanup: func(ctx context.Context) error {
			return client.Passwords.Delete(ctx, &ps.DeleteDatabaseBranchPasswordRequest{
				Organization: opt.Organization,
				Database:     opt.Database,
				Branch:       opt.Branch,
				Name:         pw.Name,
				PasswordId:   pw.PublicID,
			})
		},
		renew: func(ctx context.Context) error {
			_, err := client.Passwords.Renew(ctx, &ps.RenewDatabaseBranchPasswordRequest{
				Organization: opt.Organization,
				Database:     opt.Database,
				Branch:       opt.Branch,
				PasswordId:   pw.PublicID,
			})
			return err
		},
	}, nil
}

// GenerateName takes a given prefix and adds a randomized suffix and datetime
// marker that is useful for password Names, which will be shown in the product UI.
func GenerateName(prefix string) string {
	return fmt.Sprintf(
		"%s-%s-%s",
		prefix,
		time.Now().Format("2006-01-02"),
		nanoid.MustGenerate(publicIdAlphabet, publicIdLength),
	)
}
