package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"
)

const (
	DefaultBaseURL       = "https://planetscale.us.auth0.com/"
	formMediaType        = "application/x-www-form-urlencoded"
	jsonMediaType        = "application/json"
	DefaultAudienceURL   = "https://bb-test-api.planetscale.com"
	DefaultOAuthClientID = "ZK3V2a5UERfOlWxi5xRXrZZFmvhnf1vg"
)

// Authenticator is the interface for authentication via device oauth
type Authenticator interface {
	VerifyDevice(ctx context.Context, oauthClientID string, audienceURL string) (*DeviceVerification, error)
}

var _ Authenticator = (*DeviceAuthenticator)(nil)

// DeviceCodeResponse encapsulates the response for obtaining a device code.
type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationCompleteURI string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	PollingInterval         int    `json:"interval"`
}

// DeviceVerification represents the response from verifying a device.
type DeviceVerification struct {
	DeviceCode              string
	UserCode                string
	VerificationURL         string
	VerificationCompleteURL string
	CheckInterval           time.Duration
	ExpiresAt               time.Time
}

// DeviceAuthenticator performs the authentication flow for logging in.
type DeviceAuthenticator struct {
	client  *http.Client
	BaseURL *url.URL
}

// New returns an instance of the DeviceAuthenticator
func New(client *http.Client) (*DeviceAuthenticator, error) {
	if client == nil {
		client = cleanhttp.DefaultClient()
	}

	baseURL, err := url.Parse(DefaultBaseURL)
	if err != nil {
		return nil, err
	}
	return &DeviceAuthenticator{
		client:  client,
		BaseURL: baseURL,
	}, nil
}

// VerifyDevice performs the device verification API calls.
func (d *DeviceAuthenticator) VerifyDevice(ctx context.Context, clientID string, audienceURL string) (*DeviceVerification, error) {
	payload := strings.NewReader(fmt.Sprintf("client_id=%s&scope=profile,email,read:databases,write:databases&audience=%s", clientID, audienceURL))
	req, err := d.NewFormRequest(ctx, http.MethodPost, "oauth/device/code", payload)
	if err != nil {
		return nil, err
	}

	res, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	deviceCodeRes := &DeviceCodeResponse{}
	err = json.NewDecoder(res.Body).Decode(deviceCodeRes)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding device code response")
	}

	checkInterval := time.Duration(deviceCodeRes.PollingInterval) * time.Second
	expiresAt := time.Now().Add(time.Duration(deviceCodeRes.ExpiresIn) * time.Second)

	return &DeviceVerification{
		DeviceCode:              deviceCodeRes.DeviceCode,
		UserCode:                deviceCodeRes.UserCode,
		VerificationCompleteURL: deviceCodeRes.VerificationCompleteURI,
		VerificationURL:         deviceCodeRes.VerificationURI,
		ExpiresAt:               expiresAt,
		CheckInterval:           checkInterval,
	}, nil
}

// NewFormRequest creates a new form URL encoded request
func (d *DeviceAuthenticator) NewFormRequest(ctx context.Context, method string, path string, body io.Reader) (*http.Request, error) {
	u, err := d.BaseURL.Parse(path)
	if err != nil {
		return nil, err
	}

	var req *http.Request
	switch method {
	case http.MethodGet:
		req, err = http.NewRequest(method, u.String(), nil)
		if err != nil {
			return nil, err
		}
	default:
		req, err = http.NewRequest(method, u.String(), body)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", formMediaType)
	}

	req.Header.Set("Accept", jsonMediaType)
	req = req.WithContext(ctx)
	return req, nil
}
