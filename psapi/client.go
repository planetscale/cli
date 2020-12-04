package psapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
)

const DefaultBaseURL = "https://ps-api-bb-rails-prod.herokuapp.com"

// Client encapsulates a client that talks to the PlanetScale API
type Client struct {
	client *http.Client

	// Base URL for the API
	BaseURL *url.URL

	// WithAccessToken
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

	baseURL, _ := url.Parse(DefaultBaseURL)
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

// Do send an HTTP request
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*http.Response, error) {
	req = req.WithContext(ctx)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	// TODO(iheanyi): Add basic error response handling here.
	return res, nil
}
