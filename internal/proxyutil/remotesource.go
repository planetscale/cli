package proxyutil

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
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
	pkey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
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
		AccessHost: cert.AccessHost,
		Ports: proxy.RemotePorts{
			Proxy: cert.Ports.Proxy,
			MySQL: cert.Ports.MySQL,
		},
	}, nil
}
