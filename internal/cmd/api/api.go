package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

type ApiOpts struct {
	Method    string
	Query     []string
	Header    []string
	Field     []string
	Input     string
	ReadStdin bool
}

// ApiCmd helps users perform API calls using the CLI.
func ApiCmd(ch *cmdutil.Helper, userAgent string, defaultHeaders map[string]string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api <endpoint>",
		Short: "Performs authenticated calls against the PlanetScale API. Useful for scripting.",
		Long: heredoc.Docf(`Performs authenticated calls against the PlanetScale API and prints the response to stdout.

		ENDPOINT

			The endpoint should be a path for the v1 PlanetScale API, not including the "v1" prefix. The placeholder "{org}",
			"{db}" and "{branch}" will be replaced with the currently selected organization, and the project's database and
			branch. If invoked from outside a project, "{db}" and "{branch}" yield an empty string.

			Query parameters, such as for pagination, can be added with the --query/-Q flag, in "key=value" format.
			By default, all HTTP requests use the GET method, unless fields for the body are passed via --field/-F, in which
			case a POST method is used. If another method is required, use the --method/-X flag to specify which one to use.

		HEADERS

			Use the --header/-H flag to specify headers manually. Default headers are already included such as "User-Agent",
			"Content-Type", "Accept" and "Authorization".

		BODY

			If a body is required for the endpoint you're targeting, the --field/-F flag allows you to specify values at specific
			paths of a JSON document. Repeat the flag to set multiple values. All bodies sent are JSON objects, it isn't possible
			to send non JSON content, or JSON entities of non-object type as the root of the body.

			If you wish to pass the content of a file as body, use the --input/-I flag. If the body should be read from stdin,
			use --input="-".

			Fields specified with --field/-F will be updated in the content of --input/-I if the content is JSON. If not, an error
			will be returned.
		`),
		Example: heredoc.Docf(`
		# get the current user

		$ pscale api user

		# list an org's databases

		$ pscale api organizations/{org}/databases

		# create a database

		$ pscale api organizations/{org}/databases -F 'name="my-database"'

		# get a database

		$ pscale api organizations/{org}/databases/my-database

		# delete a database

		$ pscale api -X DELETE organizations/{org}/databases/my-database

		# add a note on a database from the content of a file

		$ pscale api -X PATCH organizations/{org}/databases/{db} -F 'notes=@mynotes.txt'

		# create a database branch from a file

		$ pscale api organizations/{org}/databases/{db}/branches --input=spec.json

		# create a database branch from stdin, override the name

		$ pscale api organizations/{org}/databases/{db}/branches --input=- -F 'name="my-name"'

		`),
		Args:              cobra.ExactArgs(1),
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.PersistentFlags().StringVar(&ch.Config.Database, "database", ch.Config.Database, "The database this project is using")
	cmd.PersistentFlags().StringVar(&ch.Config.Branch, "branch", ch.Config.Branch, "The branch this project is using")

	opts := &ApiOpts{}
	cmd.Flags().StringVarP(&opts.Method, "method", "X", "GET", "HTTP method to use for the request.  Defaults to GET for requests without a body, or POST when a body is specified with --field/-F or --input/-I")
	cmd.Flags().StringArrayVarP(&opts.Query, "query", "Q", nil, "query to append to the URL path, in `key=value` format")
	cmd.Flags().StringArrayVarP(&opts.Header, "header", "H", nil, "HTTP headers to add to the request")
	cmd.Flags().StringArrayVarP(&opts.Field, "field", "F", nil, "HTTP body to send with the request, in `key=value` format where value is a JSON entity, unless value starts with a @ in which case the string after @ represents a file that will be read. Nested types are represented as 'root.depth1.depth2=value'")
	cmd.Flags().StringVarP(&opts.Input, "input", "I", "", "HTTP body to send with the request, as a file that will be read and then sent.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		method := opts.Method
		if !cmd.Flags().Changed("method") && len(opts.Field) > 0 {
			method = http.MethodPost
		}

		u, err := parseURL(ch, opts, args[0])
		if err != nil {
			return fmt.Errorf("parsing URL: %w", err)
		}

		body, err := parseBody(opts)
		if err != nil {
			return fmt.Errorf("parsing HTTP request body: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
		if err != nil {
			return fmt.Errorf("preparing HTTP request: %w", err)
		}

		req.Header, err = parseHeader(opts, req.Method, userAgent, defaultHeaders)
		if err != nil {
			return fmt.Errorf("parsing HTTP request header: %w", err)
		}

		if ch.Debug() {
			debugReq, err := httputil.DumpRequestOut(req, true)
			if err != nil {
				return fmt.Errorf("dumping request output: %w", err)
			}
			debugReq = append(debugReq, '\n')
			_, err = os.Stderr.Write(debugReq)
			if err != nil {
				return fmt.Errorf("writing request output to stderr: %w", err)
			}
		}

		// Store the original domain for detecting cross-domain redirects
		originalDomain := extractRootDomain(req.URL.Host)
		redirectCheck := makeRedirectCheck(originalDomain)

		var cl *http.Client
		if ch.Config.AccessToken != "" {
			// Create a transport chain that uses OAuth2 token but honors our redirect policy
			tok := &oauth2.Token{AccessToken: ch.Config.AccessToken}
			cl = oauth2.NewClient(ctx, oauth2.StaticTokenSource(tok))

			// Set our custom redirect policy
			cl.CheckRedirect = makeRedirectCheck(originalDomain)
		} else if ch.Config.ServiceToken != "" && ch.Config.ServiceTokenID != "" {
			// For service tokens, manually set the auth header
			req.Header.Set("Authorization", ch.Config.ServiceTokenID+":"+ch.Config.ServiceToken)

			// Create client with our redirect policy
			cl = &http.Client{
				CheckRedirect: redirectCheck,
			}
		} else {
			cl = &http.Client{}
		}

		res, err := cl.Do(req)
		if err != nil {
			return fmt.Errorf("sending HTTP request: %w", err)
		}
		defer res.Body.Close()

		// Check if we received a redirect response
		if res.StatusCode >= 300 && res.StatusCode < 400 {
			location := res.Header.Get("Location")
			if location != "" {
				newRes, handleErr := handleRedirect(ctx, req, res, location, ch.Debug())
				if handleErr != nil {
					return handleErr
				}

				// If handleRedirect returned a new response, use it instead
				if newRes != nil {
					res.Body.Close()
					res = newRes
				}
			}
		}

		if _, err := io.Copy(os.Stdout, res.Body); err != nil {
			return fmt.Errorf("reading HTTP response body: %w", err)
		}

		if res.StatusCode > 399 {
			return fmt.Errorf("HTTP %s", res.Status)
		}

		return nil
	}

	return cmd
}

// Create a custom redirect check function that blocks cross-domain redirects.
// The OAuth2 transport adds the Authorization header to every
// request, even after a cross-domain redirect, which is really bad.
// So we detect cross-domain redirects and handle them with a
// separate, auth-less http.Client.
func makeRedirectCheck(originalDomain string) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		// Check if this is a redirect to a different domain
		currentDomain := extractRootDomain(req.URL.Host)
		if originalDomain != currentDomain {
			// For cross-domain redirects, don't follow automatically
			// We'll handle this manually without sending auth headers
			return http.ErrUseLastResponse
		}

		// Standard Go redirect policy (max 10 redirects)
		if len(via) >= 10 {
			return http.ErrUseLastResponse
		}
		return nil
	}
}

func parseURL(ch *cmdutil.Helper, opts *ApiOpts, endpoint string) (*url.URL, error) {
	reqPath := endpoint
	reqPath = strings.ReplaceAll(reqPath, `{org}`, ch.Config.Organization)
	reqPath = strings.ReplaceAll(reqPath, `{db}`, ch.Config.Database)
	reqPath = strings.ReplaceAll(reqPath, `{branch}`, ch.Config.Branch)

	u, err := url.Parse(ch.Config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing base URL: %w", err)
	}
	u = u.ResolveReference(&url.URL{Path: path.Join("v1", reqPath)})

	if len(opts.Query) > 0 {
		for _, param := range opts.Query {
			k, v, ok := strings.Cut(param, "=")
			if !ok {
				return nil, fmt.Errorf("missing '=' in query param %q", param)
			}
			q := u.Query()
			q.Add(k, v)
			u.RawQuery = q.Encode()
		}
	}
	return u, nil
}

func parseHeader(opts *ApiOpts, method, userAgent string, defaultHeaders map[string]string) (http.Header, error) {
	out := make(http.Header)
	for k, v := range defaultHeaders {
		out.Set(k, v)
	}
	out.Set("User-Agent", userAgent)
	switch method {
	case http.MethodGet:
		out.Set("Accept", "application/json")
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		out.Set("Accept", "application/json")
		out.Set("Content-Type", "application/json")
	}
	for _, header := range opts.Header {
		k, v, ok := strings.Cut(header, ":")
		if !ok {
			return nil, fmt.Errorf("invalid header: %q", header)
		}
		out.Set(k, strings.TrimPrefix(v, " "))
	}
	return out, nil
}

func parseBody(opts *ApiOpts) (io.Reader, error) {
	if opts.Input == "" && len(opts.Field) == 0 {
		return nil, nil
	}

	var (
		bodyMap map[string]interface{}
		raw     []byte
		err     error
	)
	if opts.Input != "" {
		if opts.Input == "-" {
			raw, err = io.ReadAll(os.Stdin)
			if err != nil {
				return nil, fmt.Errorf("reading body from stdin: %w", err)
			}
		} else {
			raw, err = os.ReadFile(opts.Input)
			if err != nil {
				return nil, fmt.Errorf("reading body from file %q: %w", opts.Input, err)
			}
		}
	}
	if len(raw) > 0 {
		bodyMap = make(map[string]interface{})
		if err := json.Unmarshal(raw, &bodyMap); err != nil {
			// body wasn't JSON
			if len(opts.Field) > 0 {
				return nil, fmt.Errorf("parsing input as JSON (--field/-F was specified): %w", err)
			}

			return bytes.NewReader(raw), nil
		}
		// body is JSON, continue and append to it with values in `--field`
	}

	bodyMap, err = parseFields(bodyMap, opts.Field)
	if err != nil {
		return nil, fmt.Errorf("parsing body field: %w", err)
	}
	if bodyMap != nil {
		jsonBody, err := json.MarshalIndent(bodyMap, "", "\t")
		if err != nil {
			return nil, fmt.Errorf("parsing body field: %w", err)
		}
		return bytes.NewReader(jsonBody), nil
	}
	return nil, nil
}

func parseFields(out map[string]interface{}, fields []string) (map[string]interface{}, error) {
	if out == nil {
		out = make(map[string]interface{}, len(fields))
	}
	for _, field := range fields {
		err := parseFieldInto(out, field)
		if err != nil {
			return nil, fmt.Errorf("parsing field %q: %w", field, err)
		}
	}
	return out, nil
}

func parseFieldInto(tgt map[string]interface{}, field string) error {
	k, v, ok := strings.Cut(field, "=")
	if !ok {
		return fmt.Errorf("no `=` found in field %q", field)
	}
	paths := strings.Split(k, ".")

	tail := tgt

	lastPath := paths[len(paths)-1]
	for _, path := range paths[:len(paths)-1] {
		nextTail, ok := tail[path].(map[string]interface{})
		if !ok {
			nextTail = make(map[string]interface{})
		}
		tail[path] = nextTail
		tail = nextTail
	}
	parsed, err := parseValue(v)
	if err != nil {
		return fmt.Errorf("parsing value of field %q: %w", field, err)
	}
	tail[lastPath] = parsed
	return nil
}

func parseValue(s string) (interface{}, error) {
	if len(s) > 0 && s[0] == '@' {
		filename := s[1:]
		value, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("reading value %q as file: %w", s, err)
		}
		return string(value), nil
	}

	var v interface{}
	return v, json.Unmarshal([]byte(s), &v)
}

// handleRedirect processes cross-domain redirects by following the redirect
// manually without sending authentication headers
func handleRedirect(ctx context.Context, originalReq *http.Request, originalRes *http.Response, location string, debug bool) (*http.Response, error) {
	// Create a new request without auth headers
	redirectReq, err := http.NewRequestWithContext(ctx, "GET", location, nil)
	if err != nil {
		return nil, fmt.Errorf("preparing redirect request: %w", err)
	}

	// Copy all headers except Authorization
	for k, v := range originalReq.Header {
		if !strings.EqualFold(k, "Authorization") {
			redirectReq.Header[k] = v
		}
	}

	if debug {
		fmt.Fprintf(os.Stderr, "Following cross-domain redirect to %s without auth headers\n", location)
	}

	// Use a new client without auth for the redirect
	redirectClient := &http.Client{}
	redirectRes, err := redirectClient.Do(redirectReq)
	if err != nil {
		return nil, fmt.Errorf("following redirect: %w", err)
	}

	return redirectRes, nil
}

// extractRootDomain gets the root domain from a host string.  This just
// assumes that the last two parts of the domain name are the root domain,
// which is fine for planetscale.com but would break down under compound
// TLDs like .co.uk.
func extractRootDomain(host string) string {
	// Remove port if present
	host, _, _ = strings.Cut(host, ":")

	parts := strings.Split(host, ".")
	if len(parts) <= 2 {
		return host
	}

	return strings.Join(parts[len(parts)-2:], ".")
}
