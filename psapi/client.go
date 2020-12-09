package psapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
)

const (
	DefaultBaseURL = "https://ps-api-bb-rails-prod.herokuapp.com/"
	jsonMediaType  = "application/json"
)

// Client encapsulates a client that talks to the PlanetScale API
type Client struct {
	client *http.Client

	// Base URL for the API
	BaseURL *url.URL

	Databases DatabasesService
}

// ClientOption provides a variadic option for configuring the client
type ClientOption func(c *Client) error

// SetBaseURL overrides the base URL for the API.
func SetBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		parsedURL, err := url.Parse(baseURL)
		if err != nil {
			return err
		}

		c.BaseURL = parsedURL
		return nil
	}
}

// NewClient instantiates an instance of the PlanetScale API client
func NewClient(client *http.Client, opts ...ClientOption) (*Client, error) {
	if client == nil {
		client = http.DefaultClient
	}

	baseURL, err := url.Parse(DefaultBaseURL)
	if err != nil {
		return nil, err
	}
	c := &Client{
		client:  client,
		BaseURL: baseURL,
	}

	for _, opt := range opts {
		err := opt(c)
		if err != nil {
			return nil, err
		}
	}

	c.Databases = &databasesService{
		client: c,
	}

	return c, nil
}

// NewClientFromToken instantiates an API client with a given access token.
func NewClientFromToken(accessToken string, opts ...ClientOption) (*Client, error) {
	if accessToken == "" {
		return nil, errors.New("missing access token")
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	oauthClient := oauth2.NewClient(context.Background(), tokenSource)

	return NewClient(oauthClient, opts...)
}

// GetAPIEndpoint simply returns an API endpoint.
func (c *Client) GetAPIEndpoint(path string) string {
	return fmt.Sprintf("%s/%s", c.BaseURL, path)
}

// Do executes the inputted HTTP request.
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*http.Response, error) {
	req = req.WithContext(ctx)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		errorRes := &ErrorResponse{}
		err = json.NewDecoder(res.Body).Decode(errorRes)
		if err != nil {
			return nil, err
		}

		return nil, errorRes
	}

	if v != nil {
		err = json.NewDecoder(res.Body).Decode(v)
		if err != nil {
			return nil, err
		}
	}

	// TODO(iheanyi): Add basic error response handling here.
	return res, nil
}

func (c *Client) NewRequest(method string, path string, body interface{}) (*http.Request, error) {
	u, err := c.BaseURL.Parse(path)
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
		buf := new(bytes.Buffer)
		if body != nil {
			err = json.NewEncoder(buf).Encode(body)
			if err != nil {
				return nil, err
			}
		}

		req, err = http.NewRequest(method, u.String(), buf)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", jsonMediaType)
	}

	req.Header.Set("Accept", jsonMediaType)

	return req, nil
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e ErrorResponse) Error() string {
	return e.Message
}
