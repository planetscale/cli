package proxyutil

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"

	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/planetscale/sql-proxy/proxy"
)

type RemoteCertSource struct {
	client *ps.Client
}

func NewRemoteCertSource(client *ps.Client) *RemoteCertSource {
	return &RemoteCertSource{
		client: client,
	}
}

func (r *RemoteCertSource) Cert(ctx context.Context, org, db, branch string) (*proxy.Cert, error) {
	pkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("couldn't generate private key: %s", err)
	}

	cert, err := r.client.Certificates.Create(ctx, &ps.CreateCertificateRequest{
		Organization: org,
		DatabaseName: db,
		Branch:       branch,
		PrivateKey:   pkey,
	})
	if err != nil {
		return nil, err
	}

	return &proxy.Cert{
		ClientCert: cert.ClientCert,
		CACert:     cert.CACert,
		RemoteAddr: cert.RemoteAddr,
	}, nil
}
