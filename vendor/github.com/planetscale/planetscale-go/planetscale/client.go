package planetscale

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-cleanhttp"
	"golang.org/x/oauth2"
)

const (
	DefaultBaseURL = "https://api.planetscale.com/"
	jsonMediaType  = "application/json"
)

// Client encapsulates a client that talks to the PlanetScale API
type Client struct {
	// client represents the HTTP client used for making HTTP requests.
	client *http.Client

	// base URL for the API
	baseURL *url.URL

	Databases        DatabasesService
	Certificates     CertificatesService
	DatabaseBranches DatabaseBranchesService
	Organizations    OrganizationsService
	SchemaSnapshots  SchemaSnapshotsService
	DeployRequests   DeployRequestsService
	ServiceTokens    ServiceTokenService
}

// ClientOption provides a variadic option for configuring the client
type ClientOption func(c *Client) error

// WithBaseURL overrides the base URL for the API.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		parsedURL, err := url.Parse(baseURL)
		if err != nil {
			return err
		}

		c.baseURL = parsedURL
		return nil
	}
}

// WithAccessToken configures a client with the given PlanetScale access token.
func WithAccessToken(token string) ClientOption {
	return func(c *Client) error {
		if token == "" {
			return errors.New("missing access token")
		}

		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})

		// make sure we use our own HTTP client
		ctx := context.WithValue(context.Background(), oauth2.HTTPClient, c.client)
		oauthClient := oauth2.NewClient(ctx, tokenSource)

		c.client = oauthClient
		return nil
	}
}

// WithServiceToken configures a client with the given PlanetScale Service Token
func WithServiceToken(name, token string) ClientOption {
	return func(c *Client) error {
		if token == "" || name == "" {
			return errors.New("missing token name and string")
		}

		transport := serviceTokenTransport{
			rt:        c.client.Transport,
			token:     token,
			tokenName: name,
		}

		c.client.Transport = &transport
		return nil
	}
}

// WithHTTPClient configures the PLanetScale client with the given HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) error {
		if client == nil {
			client = cleanhttp.DefaultClient()
		}

		c.client = client
		return nil
	}
}

// NewClient instantiates an instance of the PlanetScale API client.
func NewClient(opts ...ClientOption) (*Client, error) {
	baseURL, err := url.Parse(DefaultBaseURL)
	if err != nil {
		return nil, err
	}

	c := &Client{
		client:  cleanhttp.DefaultClient(),
		baseURL: baseURL,
	}

	for _, opt := range opts {
		err := opt(c)
		if err != nil {
			return nil, err
		}
	}

	c.Databases = &databasesService{client: c}
	c.Certificates = &certificatesService{client: c}
	c.DatabaseBranches = &databaseBranchesService{client: c}
	c.Organizations = &organizationsService{client: c}
	c.SchemaSnapshots = &schemaSnapshotsService{client: c}
	c.DeployRequests = &deployRequestsService{client: c}
	c.ServiceTokens = &serviceTokenService{client: c}

	return c, nil
}

// Do sends an HTTP request and returns an HTTP response with the configured
// HTTP client.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 400 {
		out, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		errorRes := &ErrorResponse{}
		err = json.Unmarshal(out, errorRes)
		if err != nil {
			return nil, err
		}

		// json.Unmarshal doesn't return an error if the response
		// body has a different protocol then "ErrorResponse". We
		// check here to make sure that errorRes is populated. If
		// not, we return the full response back to the user, so
		// they can debug the issue.
		// TODO(fatih): fix the behavior on the API side
		if *errorRes == (ErrorResponse{}) {
			return nil, errors.New(string(out))
		}

		return res, errorRes
	}

	return res, nil
}

func (c *Client) newRequest(method string, path string, body interface{}) (*http.Request, error) {
	u, err := c.baseURL.Parse(path)
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

type serviceTokenTransport struct {
	rt        http.RoundTripper
	token     string
	tokenName string
}

func (t *serviceTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", t.tokenName+":"+t.token)
	return t.rt.RoundTrip(req)
}
