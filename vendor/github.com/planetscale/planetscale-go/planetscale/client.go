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

// ErrorCode defines the code of an error.
type ErrorCode string

const (
	ErrInternal          ErrorCode = "internal"           // Internal error.
	ErrInvalid           ErrorCode = "invalid"            // Invalid operation, e.g wrong params
	ErrPermission        ErrorCode = "permission"         // Permission denied.
	ErrNotFound          ErrorCode = "not_found"          // Resource not found.
	ErrRetry             ErrorCode = "retry"              // Operation should be retried.
	ErrResponseMalformed ErrorCode = "response_malformed" // Response body is malformed.
)

// Client encapsulates a client that talks to the PlanetScale API
type Client struct {
	// client represents the HTTP client used for making HTTP requests.
	client *http.Client

	// base URL for the API
	baseURL *url.URL

	Backups          BackupsService
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

	c.Backups = &backupsService{client: c}
	c.Databases = &databasesService{client: c}
	c.Certificates = &certificatesService{client: c}
	c.DatabaseBranches = &databaseBranchesService{client: c}
	c.Organizations = &organizationsService{client: c}
	c.SchemaSnapshots = &schemaSnapshotsService{client: c}
	c.DeployRequests = &deployRequestsService{client: c}
	c.ServiceTokens = &serviceTokenService{client: c}

	return c, nil
}

// do makes an HTTP request and populates the given struct v from the response.
func (c *Client) do(ctx context.Context, req *http.Request, v interface{}) error {
	req = req.WithContext(ctx)
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return c.handleResponse(ctx, res, v)
}

// handleResponse makes an HTTP request and populates the given struct v from
// the response.  This is meant for internal testing and shouldn't be used
// directly. Instead please use `Client.do`.
func (c *Client) handleResponse(ctx context.Context, res *http.Response, v interface{}) error {
	out, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode >= 400 {
		// errorResponse represents an error response from the API
		type errorResponse struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}

		errorRes := &errorResponse{}
		err = json.Unmarshal(out, errorRes)
		if err != nil {
			if _, ok := err.(*json.SyntaxError); ok {
				return &Error{
					msg:  "malformed response body received",
					Code: ErrResponseMalformed,
					Meta: map[string]string{
						"body": string(out),
						"err":  err.Error(),
					},
				}
			}
			return err
		}

		// json.Unmarshal doesn't return an error if the response
		// body has a different protocol then "ErrorResponse". We
		// check here to make sure that errorRes is populated. If
		// not, we return the full response back to the user, so
		// they can debug the issue.
		// TODO(fatih): fix the behavior on the API side
		if *errorRes == (errorResponse{}) {
			return &Error{
				msg:  "internal error, please open an issue to github.com/planetscale/planetscale-go",
				Code: ErrInternal,
				Meta: map[string]string{
					"body": string(out),
				},
			}
		}

		var errCode ErrorCode
		switch errorRes.Code {
		case "not_found":
			errCode = ErrNotFound
		case "unauthorized":
			errCode = ErrPermission
		case "invalid_params":
			errCode = ErrInvalid
		case "unprocessable":
			errCode = ErrRetry
		}

		return &Error{
			msg:  errorRes.Message,
			Code: errCode,
		}
	}

	// this means we don't care about unmrarshaling the response body into v
	if v == nil {
		return nil
	}

	err = json.Unmarshal(out, &v)
	if err != nil {
		if _, ok := err.(*json.SyntaxError); ok {
			return &Error{
				msg:  "malformed response body received",
				Code: ErrResponseMalformed,
				Meta: map[string]string{
					"body": string(out),
				},
			}
		}
		return err
	}

	return nil
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

type serviceTokenTransport struct {
	rt        http.RoundTripper
	token     string
	tokenName string
}

func (t *serviceTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", t.tokenName+":"+t.token)
	return t.rt.RoundTrip(req)
}

// Error represents common errors originating from the Client.
type Error struct {
	// msg contains the human readable string
	msg string

	// Code specifies the error code. i.e; NotFound, RateLimited, etc...
	Code ErrorCode

	// Meta contains additional information depending on the error code. As an
	// example, if the Code is "ErrResponseMalformed", the map will be: ["body"]
	// = "body of the response"
	Meta map[string]string
}

// Error returns the string representation of the error.
func (e *Error) Error() string { return e.msg }
