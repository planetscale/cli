package proxyutil

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"github.com/planetscale/cli/internal/cmdutil"
	"time"

	nanoid "github.com/matoous/go-nanoid/v2"

	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/planetscale/sql-proxy/proxy"
)

type RemoteCertSource struct {
	client *ps.Client
	role   string
}

func NewRemoteCertSource(client *ps.Client, role cmdutil.PasswordRole) *RemoteCertSource {
	return &RemoteCertSource{
		client: client,
		role:   role.ToString(),
	}
}

const publicIdAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
const publicIdLength = 6

func (r *RemoteCertSource) Cert(ctx context.Context, org, db, branch string) (*proxy.Cert, error) {
	pkey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("couldn't generate private key: %s", err)
	}

	request := &ps.DatabaseBranchCertificateRequest{
		Organization: org,
		Database:     db,
		Role:         r.role,
		Branch:       branch,
		DisplayName:  fmt.Sprintf("pscale-cli-%s-%s", time.Now().Format("2006-01-02"), nanoid.MustGenerate(publicIdAlphabet, publicIdLength)),
		PrivateKey:   pkey,
	}

	cert, err := r.client.Certificates.Create(ctx, request)
	if err != nil {
		return nil, err
	}

	tlsPair, err := cert.X509KeyPair(request)
	if err != nil {
		return nil, err
	}

	return &proxy.Cert{
		ClientCert: tlsPair,
		AccessHost: cert.Branch.AccessHostURL,
		Ports: proxy.RemotePorts{
			Proxy: 3307,
			MySQL: 3306,
		},
	}, nil
}
