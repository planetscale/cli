// Package connections is a branch-connections HTTP client that follows the
// patterns of github.com/planetscale/planetscale-go: a typed Client over
// shared auth, base-URL, and transport machinery. The auth header
// construction, error formatting, and response decode helpers mirror the
// equivalents in planetscale-go.
package connections

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	clientUserAgent = "pscale-cli"
	defaultTimeout  = 5 * time.Second

	// Retry defaults for list-only 503s returned while the server's
	// connection snapshot cache is being populated. The floor is 50ms
	// because the server typically repopulates that cache in tens of
	// milliseconds; retrying at 100ms would routinely wait 50ms after the
	// cache is already warm, while 50ms catches it shortly after the lock
	// releases. Jitter (25ms) defends against lock-step retries across
	// multiple concurrent CLI users on the same branch. Budget (2s) caps
	// total perceived list latency before surfacing a friendly warming
	// message: ~2x the polling cadence, so a single hiccup doesn't surface
	// as an immediate user-visible failure.
	defaultListRetryBudget  = 2 * time.Second
	defaultListRetryBackoff = 50 * time.Millisecond
	defaultListRetryJitter  = 25 * time.Millisecond
)

var (
	errListWarming          = errors.New("list connections: server warming")
	errListWarmingExhausted = errors.New("list connections: server is warming up, please retry in a moment")
	errListInvalidResponse  = errors.New("list connections: received an invalid response, please retry")
)

type ClientConfig struct {
	BaseURL        string
	Organization   string
	Database       string
	Branch         string
	Keyspace       string
	Shard          string
	AccessToken    string
	ServiceTokenID string
	ServiceToken   string
	HTTPClient     *http.Client
	RequestTimeout time.Duration
}

type Client struct {
	cfg          ClientConfig
	client       *http.Client
	retryBudget  time.Duration
	retryBackoff time.Duration
	retryJitter  time.Duration
}

type AvailableTargets struct {
	Keyspaces []string `json:"keyspaces,omitempty"`
	Shards    []string `json:"shards,omitempty"`
}

type HTTPError struct {
	Op         string
	StatusCode int
	Message    string
	Available  AvailableTargets
}

func (e *HTTPError) Error() string {
	message := e.Message
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	detail := fmt.Sprintf("%s: HTTP %d: %s", e.Op, e.StatusCode, message)
	if len(e.Available.Keyspaces) > 0 {
		detail += fmt.Sprintf(" (available keyspaces: %s)", strings.Join(e.Available.Keyspaces, ", "))
	}
	if len(e.Available.Shards) > 0 {
		detail += fmt.Sprintf(" (available shards: %s)", strings.Join(e.Available.Shards, ", "))
	}
	return detail
}

func UserFacingError(err error, action string) error {
	if err == nil {
		return nil
	}
	return errors.New(UserFacingErrorText(err, action))
}

func UserFacingErrorText(err error, action string) string {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusForbidden {
		return fmt.Sprintf("permission denied: you don't have permission to %s live connections", action)
	}
	return err.Error()
}

func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.Organization == "" || cfg.Database == "" || cfg.Branch == "" {
		return nil, errors.New("organization, database, and branch are required")
	}
	if cfg.AccessToken == "" && (cfg.ServiceTokenID == "" || cfg.ServiceToken == "") {
		return nil, errors.New("not authenticated: provide AccessToken or ServiceTokenID/ServiceToken")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = defaultTimeout
	}
	return &Client{
		cfg:          cfg,
		client:       client,
		retryBudget:  defaultListRetryBudget,
		retryBackoff: defaultListRetryBackoff,
		retryJitter:  defaultListRetryJitter,
	}, nil
}

type listResponse struct {
	DatabaseKind DatabaseKind   `json:"database_kind"`
	CapturedAt   time.Time      `json:"captured_at"`
	Instances    []InstanceMeta `json:"instances"`
	Topology     *Topology      `json:"topology"`
	Data         []listEntry    `json:"data"`
}

type listEntry struct {
	PID             int        `json:"pid"`
	Instance        string     `json:"instance"`
	DatabaseName    string     `json:"datname"`
	Username        string     `json:"usename"`
	ApplicationName string     `json:"application_name"`
	ClientAddr      string     `json:"client_addr"`
	State           string     `json:"state"`
	WaitEventType   string     `json:"wait_event_type"`
	WaitEvent       string     `json:"wait_event"`
	BackendType     string     `json:"backend_type"`
	XactStart       *time.Time `json:"xact_start"`
	QueryStart      *time.Time `json:"query_start"`
	ConnectionID    *string    `json:"connection_id"`
	TransactionID   *string    `json:"transaction_id"`
	QueryID         *string    `json:"query_id"`
	DurationMS      int64      `json:"duration_ms"`
	BlockedBy       []int      `json:"blocked_by"`
	QueryText       string     `json:"query_text"`
}

func (e listEntry) connection() Connection {
	return Connection{
		PID:             e.PID,
		Instance:        e.Instance,
		DatabaseName:    e.DatabaseName,
		Username:        e.Username,
		ApplicationName: e.ApplicationName,
		ClientAddr:      e.ClientAddr,
		State:           e.State,
		WaitEventType:   e.WaitEventType,
		WaitEvent:       e.WaitEvent,
		BackendType:     e.BackendType,
		XactStart:       e.XactStart,
		QueryStart:      e.QueryStart,
		ConnectionID:    e.ConnectionID,
		TransactionID:   e.TransactionID,
		QueryID:         e.QueryID,
		Duration:        time.Duration(e.DurationMS) * time.Millisecond,
		BlockedBy:       e.BlockedBy,
		QueryText:       e.QueryText,
	}
}

func (s *Client) List(ctx context.Context, sort SortMode) (ConnectionList, error) {
	callerCtx := ctx
	if s.cfg.RequestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.cfg.RequestTimeout)
		defer cancel()
	}

	deadline := time.Now().Add(s.retryBudget)
	for {
		list, retryAfter, err := s.tryList(ctx, sort)
		if err == nil {
			return list, nil
		}
		if isTimeoutError(err) && ctx.Err() != nil {
			if callerCtx.Err() != nil {
				return ConnectionList{}, fmt.Errorf("list connections: %w", callerCtx.Err())
			}
			return ConnectionList{}, fmt.Errorf("list connections: request timed out after %s, please retry", s.cfg.RequestTimeout)
		}
		if !errors.Is(err, errListWarming) {
			return ConnectionList{}, err
		}
		if s.retryBudget <= 0 {
			return ConnectionList{}, errListWarmingExhausted
		}

		delay := retryAfter
		if delay <= 0 {
			delay = s.retryDelay()
		}
		if time.Now().Add(delay).After(deadline) {
			return ConnectionList{}, errListWarmingExhausted
		}

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ConnectionList{}, fmt.Errorf("list connections: %w", ctx.Err())
		}
	}
}

func (s *Client) tryList(ctx context.Context, sort SortMode) (ConnectionList, time.Duration, error) {
	resp, err := s.do(ctx, http.MethodGet, s.connectionsURL(""))
	if err != nil {
		return ConnectionList{}, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusServiceUnavailable {
		_, _ = io.Copy(io.Discard, resp.Body)
		return ConnectionList{}, parseRetryAfter(resp.Header.Get("Retry-After")), errListWarming
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ConnectionList{}, 0, fmt.Errorf("list connections: read body: %w", err)
	}

	if resp.StatusCode > 299 {
		return ConnectionList{}, 0, s.formatHTTPError("list connections", resp.StatusCode, body)
	}

	var listed listResponse
	if err := json.Unmarshal(body, &listed); err != nil {
		return ConnectionList{}, 0, errListInvalidResponse
	}

	capturedAt := listed.CapturedAt
	if capturedAt.IsZero() {
		return ConnectionList{}, 0, errors.New("list connections: response missing captured_at")
	}

	list := listed.connectionList(sort)
	if len(list.Instances) > 0 {
		roles := make(map[string]string, len(list.Instances))
		for _, m := range list.Instances {
			roles[m.ID] = m.Role
		}
		for i := range list.Connections {
			list.Connections[i].InstanceRole = roles[list.Connections[i].Instance]
		}
	}
	return list, 0, nil
}

func (r listResponse) connectionList(sort SortMode) ConnectionList {
	list := NewConnectionList(r.CapturedAt, connectionsFromEntries(r.Data), sort)
	list.DatabaseKind = r.DatabaseKind
	list.Instances = r.Instances
	list.Topology = r.Topology
	return list
}

func connectionsFromEntries(entries []listEntry) []Connection {
	connections := make([]Connection, 0, len(entries))
	for _, entry := range entries {
		connections = append(connections, entry.connection())
	}
	return connections
}

func isTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr interface{ Timeout() bool }
	return errors.As(err, &netErr) && netErr.Timeout()
}

func (s *Client) retryDelay() time.Duration {
	if s.retryJitter <= 0 {
		return s.retryBackoff
	}
	return s.retryBackoff + rand.N(s.retryJitter)
}

func parseRetryAfter(h string) time.Duration {
	h = strings.TrimSpace(h)
	if h == "" {
		return 0
	}
	if isPlainDecimalSeconds(h) {
		seconds, err := strconv.ParseFloat(h, 64)
		if err == nil && seconds > 0 {
			return time.Duration(seconds * float64(time.Second))
		}
	}
	t, err := http.ParseTime(h)
	if err != nil {
		return 0
	}
	d := time.Until(t)
	if d <= 0 {
		return 0
	}
	return d
}

func isPlainDecimalSeconds(s string) bool {
	seenDot := false
	digitsBeforeDot := 0
	digitsAfterDot := 0
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			if seenDot {
				digitsAfterDot++
			} else {
				digitsBeforeDot++
			}
		case r == '.':
			if seenDot {
				return false
			}
			seenDot = true
		default:
			return false
		}
	}
	if digitsBeforeDot == 0 {
		return false
	}
	return !seenDot || digitsAfterDot > 0
}

// CancelQuery asks the backend to cancel the active query identified by
// target.QueryID.
func (s *Client) CancelQuery(ctx context.Context, target ActionTarget) error {
	_, err := s.CancelQueryResult(ctx, target)
	return err
}

func (s *Client) CancelQueryResult(ctx context.Context, target ActionTarget) (ActionResult, error) {
	if target.QueryID == nil || *target.QueryID == "" {
		return ActionResult{}, errors.New("cancel query: query_id is required")
	}
	suffix := fmt.Sprintf("query/%s", url.PathEscape(*target.QueryID))
	return s.deleteAction(ctx, "cancel query", suffix)
}

// TerminateTransaction asks the backend to terminate the connection only if
// its current transaction matches target.TransactionID — i.e., reject if the
// connection has moved on. Same-transaction semantics live on the server.
func (s *Client) TerminateTransaction(ctx context.Context, target ActionTarget) error {
	_, err := s.TerminateTransactionResult(ctx, target)
	return err
}

// TerminateTransactionResult asks the backend to terminate a transaction and
// returns any action metadata the backend provides.
func (s *Client) TerminateTransactionResult(ctx context.Context, target ActionTarget) (ActionResult, error) {
	if target.TransactionID == nil || *target.TransactionID == "" {
		return ActionResult{}, errors.New("terminate transaction: transaction_id is required")
	}
	suffix := fmt.Sprintf("transaction/%s", url.PathEscape(*target.TransactionID))
	return s.deleteAction(ctx, "terminate transaction", suffix)
}

// TerminateConnection force-terminates the connection identified by
// target.ConnectionID without regard to its current query or transaction.
// Operator-initiated; gated by a y/n confirmation in the TUI.
func (s *Client) TerminateConnection(ctx context.Context, target ActionTarget) error {
	_, err := s.TerminateConnectionResult(ctx, target)
	return err
}

func (s *Client) TerminateConnectionResult(ctx context.Context, target ActionTarget) (ActionResult, error) {
	if target.ConnectionID == nil || *target.ConnectionID == "" {
		return ActionResult{}, errors.New("terminate connection: connection_id is required")
	}
	suffix := fmt.Sprintf("connection/%s", url.PathEscape(*target.ConnectionID))
	return s.deleteAction(ctx, "terminate connection", suffix)
}

type ActionResult struct {
	Success  bool   `json:"success"`
	Keyspace string `json:"keyspace,omitempty"`
	Shard    string `json:"shard,omitempty"`
	Tablet   string `json:"tablet,omitempty"`
	ID       int64  `json:"id,omitempty"`
	Kind     string `json:"kind,omitempty"`
}

// deleteAction issues a typed DELETE against the connection path with the
// caller-supplied suffix and returns a sanitized error on non-2xx.
func (s *Client) deleteAction(ctx context.Context, op, suffix string) (ActionResult, error) {
	if s.cfg.RequestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.cfg.RequestTimeout)
		defer cancel()
	}
	resp, err := s.do(ctx, http.MethodDelete, s.connectionsURL(suffix))
	if err != nil {
		return ActionResult{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ActionResult{}, fmt.Errorf("%s: read body: %w", op, err)
	}
	if resp.StatusCode > 299 {
		if resp.StatusCode == http.StatusNotFound {
			if idName := actionIDName(op); idName != "" {
				return ActionResult{}, fmt.Errorf("%s: %s not found; run connections show again and use a current %s", op, idName, idName)
			}
		}
		if resp.StatusCode == http.StatusUnprocessableEntity {
			if message := httpErrorEnvelopeMessage(body); message != "" {
				return ActionResult{}, fmt.Errorf("%s: %s", op, message)
			}
		}
		return ActionResult{}, s.formatHTTPError(op, resp.StatusCode, body)
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return ActionResult{Success: true}, nil
	}
	var result ActionResult
	if err := json.Unmarshal(body, &result); err != nil {
		return ActionResult{}, fmt.Errorf("%s: invalid response", op)
	}
	if !result.Success {
		return ActionResult{}, fmt.Errorf("%s: action did not succeed", op)
	}
	return result, nil
}

func actionIDName(op string) string {
	switch op {
	case "cancel query":
		return "query_id"
	case "terminate transaction":
		return "transaction_id"
	case "terminate connection":
		return "connection_id"
	default:
		return ""
	}
}

func (s *Client) formatHTTPError(op string, status int, body []byte) error {
	if status == http.StatusNotFound {
		hint := fmt.Sprintf(
			"verify --org=%q database=%q branch=%q exists and live connections are enabled",
			s.cfg.Organization, s.cfg.Database, s.cfg.Branch,
		)
		return fmt.Errorf("%s: not found (%s)", op, hint)
	}
	if status == http.StatusTooManyRequests {
		return fmt.Errorf("%s: rate limited, please retry in a moment", op)
	}
	if status >= 500 {
		return fmt.Errorf("%s: HTTP %d: %s", op, status, http.StatusText(status))
	}
	message, available := httpErrorDetails(body)
	return &HTTPError{
		Op:         op,
		StatusCode: status,
		Message:    message,
		Available:  available,
	}
}

func httpErrorDetails(body []byte) (string, AvailableTargets) {
	var envelope struct {
		Message   string           `json:"message"`
		Available AvailableTargets `json:"available"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil {
		return envelope.Message, envelope.Available
	}
	if detail := nonJSONBody(body); detail != "" {
		return detail, AvailableTargets{}
	}
	return "", AvailableTargets{}
}

func nonJSONBody(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return ""
	}
	if trimmed[0] == '{' || trimmed[0] == '[' {
		return ""
	}
	return trimmed
}

func httpErrorEnvelopeMessage(body []byte) string {
	var envelope struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil {
		return envelope.Message
	}
	return ""
}

// UnknownInstanceError reports an --instance filter value that matches none of
// the instances in the list response.
type UnknownInstanceError struct {
	Instance string
	Valid    []string
}

func (e *UnknownInstanceError) Error() string {
	return fmt.Sprintf("unknown instance %q (valid instances: %s)", e.Instance, strings.Join(e.Valid, ", "))
}

func (s *Client) connectionsURL(suffix string) string {
	raw := fmt.Sprintf(
		"%s/v1/organizations/%s/databases/%s/branches/%s/connections",
		strings.TrimRight(s.cfg.BaseURL, "/"),
		url.PathEscape(s.cfg.Organization),
		url.PathEscape(s.cfg.Database),
		url.PathEscape(s.cfg.Branch),
	)
	if suffix != "" {
		raw += "/" + suffix
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	if s.cfg.Keyspace != "" {
		q.Set("keyspace", s.cfg.Keyspace)
	}
	if s.cfg.Shard != "" {
		q.Set("shard", s.cfg.Shard)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *Client) do(ctx context.Context, method, urlStr string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", clientUserAgent)
	req.Header.Set("Accept", "application/json")
	if s.cfg.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.AccessToken)
	} else {
		req.Header.Set("Authorization", s.cfg.ServiceTokenID+":"+s.cfg.ServiceToken)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	return resp, nil
}
