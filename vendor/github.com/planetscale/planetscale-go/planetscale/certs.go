package planetscale

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
)

type CreateCertificateRequest struct {
	Organization string
	DatabaseName string
	Branch       string

	// PrivateKey is used to sign the Certificate Sign Request (CSR).
	PrivateKey *rsa.PrivateKey
}

type CertificatesService interface {
	Create(context.Context, *CreateCertificateRequest) (*Cert, error)
}

type Cert struct {
	ClientCert tls.Certificate
	CACert     *x509.Certificate
	RemoteAddr string
}

type certificatesService struct {
	client *Client
}

var _ CertificatesService = &certificatesService{}

func NewCertsService(client *Client) *certificatesService {
	return &certificatesService{
		client: client,
	}
}

func (c *certificatesService) Create(ctx context.Context, r *CreateCertificateRequest) (*Cert, error) {
	cn := fmt.Sprintf("%s/%s/%s", r.Organization, r.DatabaseName, r.Branch)
	subj := pkix.Name{
		CommonName: cn,
	}

	template := x509.CertificateRequest{
		Version:            1,
		Subject:            subj,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &template, r.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("unable to create csr: %s", err)
	}

	var buf bytes.Buffer
	err = pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})
	if err != nil {
		return nil, fmt.Errorf("unable to encode the CSR to PEM: %s", err)
	}

	var certReq = struct {
		CSR string `json:"csr"`
	}{
		CSR: buf.String(),
	}

	req, err := c.client.newRequest(
		http.MethodPost,
		fmt.Sprintf("%s/%s/branches/%s/create-certificate",
			databasesAPIPath(r.Organization),
			r.DatabaseName,
			r.Branch,
		),
		certReq,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating request for create certificates: %s", err)
	}

	res, err := c.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var cr struct {
		Certificate      string `json:"certificate"`
		CertificateChain string `json:"certificate_chain"`
		RemoteAddr       string `json:"remote_addr"`
	}

	err = json.NewDecoder(res.Body).Decode(&cr)
	if err != nil {
		return nil, err
	}

	caCert, err := parseCert(cr.CertificateChain)
	if err != nil {
		return nil, fmt.Errorf("parsing certificate chain failed: %s", err)
	}

	privateKey := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(r.PrivateKey),
		},
	)

	clientCert, err := tls.X509KeyPair([]byte(cr.Certificate), privateKey)
	if err != nil {
		return nil, fmt.Errorf("parsing client certificate failed: %s", err)
	}

	return &Cert{
		ClientCert: clientCert,
		CACert:     caCert,
		RemoteAddr: cr.RemoteAddr,
	}, nil
}

func parseCert(pemCert string) (*x509.Certificate, error) {
	bl, _ := pem.Decode([]byte(pemCert))
	if bl == nil {
		return nil, errors.New("invalid PEM: " + pemCert)
	}
	return x509.ParseCertificate(bl.Bytes)
}
