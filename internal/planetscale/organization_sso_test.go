package planetscale

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
)

func TestOrganizationSSO_Get(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/sso")
		w.WriteHeader(200)
		_, err := w.Write([]byte(`{
			"id": "org_1",
			"type": "OrganizationSSO",
			"enabled": true,
			"configured": false,
			"directory": false,
			"has_verified_domain": true,
			"domain_verification_url": null,
			"domains": [{
				"id": "dom_1",
				"type": "OrganizationDomain",
				"domain": "example.com",
				"state": "verified",
				"verified_at": "2021-01-14T10:19:23.000Z",
				"failure_reason": null,
				"created_at": "2021-01-14T10:19:23.000Z",
				"updated_at": "2021-01-14T10:19:23.000Z"
			}]
		}`))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	sso, err := client.OrganizationSSO.Get(context.Background(), &OrganizationSSORequest{Organization: "my-org"})
	c.Assert(err, qt.IsNil)
	c.Assert(sso.Enabled, qt.IsTrue)
	c.Assert(sso.Configured, qt.IsFalse)
	c.Assert(sso.HasVerifiedDomain, qt.IsTrue)
	c.Assert(sso.DomainVerificationURL, qt.IsNil)
	c.Assert(sso.Domains, qt.HasLen, 1)
	c.Assert(sso.Domains[0].Domain, qt.Equals, "example.com")
	c.Assert(sso.Domains[0].VerifiedAt, qt.Not(qt.IsNil))
	c.Assert(*sso.Domains[0].VerifiedAt, qt.Equals, time.Date(2021, time.January, 14, 10, 19, 23, 0, time.UTC))
}

func TestOrganizationSSO_Enable(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/sso")
		w.WriteHeader(200)
		_, err := w.Write([]byte(`{
			"id": "org_1",
			"type": "OrganizationSSO",
			"enabled": true,
			"configured": false,
			"directory": false,
			"has_verified_domain": false,
			"domain_verification_url": "https://portal.example/verify",
			"domains": []
		}`))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	sso, err := client.OrganizationSSO.Enable(context.Background(), &OrganizationSSORequest{Organization: "my-org"})
	c.Assert(err, qt.IsNil)
	c.Assert(sso.Enabled, qt.IsTrue)
	c.Assert(sso.DomainVerificationURL, qt.Not(qt.IsNil))
	c.Assert(*sso.DomainVerificationURL, qt.Equals, "https://portal.example/verify")
}

func TestOrganizationSSO_Configure(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/sso/configure")
		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"portal_url":"https://portal.example/sso"}`))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	portal, err := client.OrganizationSSO.Configure(context.Background(), &OrganizationSSORequest{Organization: "my-org"})
	c.Assert(err, qt.IsNil)
	c.Assert(portal.PortalURL, qt.Equals, "https://portal.example/sso")
}

func TestOrganizationSSO_ListDomains(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodGet)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/sso/domains")
		w.WriteHeader(200)
		_, err := w.Write([]byte(`[{
			"id": "dom_1",
			"type": "OrganizationDomain",
			"domain": "example.com",
			"state": "pending",
			"verified_at": null,
			"failure_reason": null,
			"created_at": "2021-01-14T10:19:23.000Z",
			"updated_at": "2021-01-14T10:19:23.000Z"
		}]`))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	domains, err := client.OrganizationSSO.ListDomains(context.Background(), &OrganizationSSORequest{Organization: "my-org"})
	c.Assert(err, qt.IsNil)
	c.Assert(domains, qt.HasLen, 1)
	c.Assert(domains[0].ID, qt.Equals, "dom_1")
	c.Assert(domains[0].State, qt.Equals, "pending")
}

func TestOrganizationSSO_Disable(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/sso")
		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"id":"org_1","type":"OrganizationSSO","enabled":false,"configured":false,"directory":false,"has_verified_domain":false,"domains":[],"domain_verification_url":null}`))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	sso, err := client.OrganizationSSO.Disable(context.Background(), &OrganizationSSORequest{Organization: "my-org"})
	c.Assert(err, qt.IsNil)
	c.Assert(sso.Enabled, qt.IsFalse)
}

func TestOrganizationSSO_EnableDirectory(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodPost)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/sso/directory")
		w.WriteHeader(200)
		_, err := w.Write([]byte(`{"portal_url":"https://portal.example/dsync"}`))
		c.Assert(err, qt.IsNil)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	portal, err := client.OrganizationSSO.EnableDirectory(context.Background(), &OrganizationSSORequest{Organization: "my-org"})
	c.Assert(err, qt.IsNil)
	c.Assert(portal.PortalURL, qt.Equals, "https://portal.example/dsync")
}

func TestOrganizationSSO_DeleteDomain(t *testing.T) {
	c := qt.New(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.Method, qt.Equals, http.MethodDelete)
		c.Assert(r.URL.Path, qt.Equals, "/v1/organizations/my-org/sso/domains/dom_1")
		w.WriteHeader(204)
	}))

	client, err := NewClient(WithBaseURL(ts.URL))
	c.Assert(err, qt.IsNil)

	err = client.OrganizationSSO.DeleteDomain(context.Background(), &DeleteOrganizationSSODomainRequest{
		Organization: "my-org",
		DomainID:     "dom_1",
	})
	c.Assert(err, qt.IsNil)
}
