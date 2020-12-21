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

	"github.com/benbjohnson/clock"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"
)

const (
	DefaultBaseURL    = "https://auth.planetscaledb.io/"
	OAuthClientID     = "dPLmLcw0S5pmeWeSRWxXxgsD8tG5Tzjj5ziMsbUKym8"
	OAuthClientSecret = "YTiMkrVjxQXUnvTA1sGu3MnIS0m05NZ6aQyuUOXaX5Y"

	formMediaType = "application/x-www-form-urlencoded"
	jsonMediaType = "application/json"
)

// Authenticator is the interface for authentication via device oauth
type Authenticator interface {
	VerifyDevice(ctx context.Context) (*DeviceVerification, error)
	GetAccessTokenForDevice(ctx context.Context, v *DeviceVerification) (string, error)
	RevokeToken(ctx context.Context, token string) error
}

var _ Authenticator = (*DeviceAuthenticator)(nil)

type AuthenticatorOption func(c *DeviceAuthenticator) error

// SetBaseURL overrides the base URL for the DeviceAuthenticator.
func SetBaseURL(baseURL string) AuthenticatorOption {
	return func(d *DeviceAuthenticator) error {
		parsedURL, err := url.Parse(baseURL)
		if err != nil {
			return err
		}

		d.BaseURL = parsedURL
		return nil
	}
}

// WithMockClock replaces the clock on the authenticator with a mock clock.
func WithMockClock(mock *clock.Mock) AuthenticatorOption {
	return func(d *DeviceAuthenticator) error {
		d.Clock = mock
		return nil
	}
}

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

// ErrorResponse is an error response from the API.
type ErrorResponse struct {
	ErrorCode   string `json:"error"`
	Description string `json:"error_description"`
}

func (e ErrorResponse) Error() string {
	return e.Description
}

// DeviceAuthenticator performs the authentication flow for logging in.
type DeviceAuthenticator struct {
	client       *http.Client
	BaseURL      *url.URL
	Clock        clock.Clock
	ClientID     string
	ClientSecret string
}

// New returns an instance of the DeviceAuthenticator
func New(client *http.Client, clientID string, clientSecret string, opts ...AuthenticatorOption) (*DeviceAuthenticator, error) {
	if client == nil {
		client = cleanhttp.DefaultClient()
	}

	baseURL, err := url.Parse(DefaultBaseURL)
	if err != nil {
		return nil, err
	}

	authenticator := &DeviceAuthenticator{
		client:       client,
		BaseURL:      baseURL,
		Clock:        clock.New(),
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	for _, opt := range opts {
		err := opt(authenticator)
		if err != nil {
			return nil, err
		}
	}

	return authenticator, nil
}

// VerifyDevice performs the device verification API calls.
func (d *DeviceAuthenticator) VerifyDevice(ctx context.Context) (*DeviceVerification, error) {
	payload := strings.NewReader(fmt.Sprintf("client_id=%s&scope=read_databases,write_databases", d.ClientID))
	req, err := d.NewFormRequest(ctx, http.MethodPost, "oauth/authorize_device", payload)
	if err != nil {
		return nil, err
	}

	res, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if _, err = checkErrorResponse(res); err != nil {
		return nil, err
	}

	deviceCodeRes := &DeviceCodeResponse{}
	err = json.NewDecoder(res.Body).Decode(deviceCodeRes)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding device code response")
	}

	checkInterval := time.Duration(deviceCodeRes.PollingInterval) * time.Second
	expiresAt := d.Clock.Now().Add(time.Duration(deviceCodeRes.ExpiresIn) * time.Second)

	return &DeviceVerification{
		DeviceCode:              deviceCodeRes.DeviceCode,
		UserCode:                deviceCodeRes.UserCode,
		VerificationCompleteURL: deviceCodeRes.VerificationCompleteURI,
		VerificationURL:         deviceCodeRes.VerificationURI,
		ExpiresAt:               expiresAt,
		CheckInterval:           checkInterval,
	}, nil
}

// GetAccessTokenForDevice uses the device verification response to fetch an
// access token.
func (d *DeviceAuthenticator) GetAccessTokenForDevice(ctx context.Context, v *DeviceVerification) (string, error) {
	var accessToken string
	var err error

	for {
		time.Sleep(v.CheckInterval)
		accessToken, err = d.requestToken(ctx, v.DeviceCode, d.ClientID)
		if accessToken == "" && err == nil {
			if d.Clock.Now().After(v.ExpiresAt) {
				err = errors.New("authentication timed out")
			} else {
				continue
			}
		}

		break
	}
	return accessToken, err
}

// OAuthTokenResponse contains the information returned after fetching an access
// token for a device.
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (d *DeviceAuthenticator) requestToken(ctx context.Context, deviceCode string, clientID string) (string, error) {
	payload := strings.NewReader(fmt.Sprintf("grant_type=device_code&device_code=%s&client_id=%s", deviceCode, clientID))
	req, err := d.NewFormRequest(ctx, http.MethodPost, "oauth/token", payload)
	if err != nil {
		return "", errors.Wrap(err, "error creating request")
	}

	res, err := d.client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "error performing http request")
	}

	defer res.Body.Close()

	isRetryable, err := checkErrorResponse(res)
	if err != nil {
		return "", err
	}

	// Bail early so the token fetching is retried.
	if isRetryable {
		return "", nil
	}

	tokenRes := &OAuthTokenResponse{}

	err = json.NewDecoder(res.Body).Decode(tokenRes)
	if err != nil {
		return "", errors.Wrap(err, "error decoding token response")
	}

	return tokenRes.AccessToken, nil
}

// RevokeToken revokes an access token.
func (d *DeviceAuthenticator) RevokeToken(ctx context.Context, token string) error {
	payload := strings.NewReader(fmt.Sprintf("client_id=%s&client_secret=%s&token=%s", d.ClientID, d.ClientSecret, token))
	req, err := d.NewFormRequest(ctx, http.MethodPost, "oauth/revoke", payload)
	if err != nil {
		return errors.Wrap(err, "error creating request")
	}

	res, err := d.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "error performing http request")
	}

	defer res.Body.Close()

	if _, err = checkErrorResponse(res); err != nil {
		return err
	}
	return nil
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

// checkErrorResponse returns whether the error is retryable or not and the
// error itself.
func checkErrorResponse(res *http.Response) (bool, error) {
	if res.StatusCode >= 400 {
		errorRes := &ErrorResponse{}
		err := json.NewDecoder(res.Body).Decode(errorRes)
		if err != nil {
			return false, errors.Wrap(err, "error decoding error response")
		}

		// If we're polling and haven't authorized yet or we need to slow down, we
		// don't wanna terminate the polling
		if errorRes.ErrorCode == "authorization_pending" || errorRes.ErrorCode == "slow_down" {
			return true, nil
		}

		return false, errorRes
	}

	return false, nil
}
