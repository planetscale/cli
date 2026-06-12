package connections

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/AlecAivazis/survey/v2"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/planetscale/cli/internal/cmdutil"
	live "github.com/planetscale/cli/internal/connections"
	"github.com/planetscale/cli/internal/connections/history"
	"github.com/planetscale/cli/internal/connections/tui"
	"github.com/planetscale/cli/internal/printer"
	"github.com/planetscale/cli/internal/promptutil"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

func TopCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags topFlags

	cmd := &cobra.Command{
		Use:   "top [database] [branch]",
		Short: "Show live branch connection activity",
		Long: `Show live branch connection activity.

Run interactively in a terminal to launch the TUI. Pipe or redirect output to
run headlessly with --capture; --duration bounds either mode and without
--duration the command runs until interrupted. Pass --replay FILE to render a
previously captured trace in the TUI — actions are rejected in replay mode.

For Postgres, connections top shows session activity across instances. For
Vitess, pass --keyspace and --shard or run interactively to select them when
the server reports available targets.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if flags.replay != "" {
				return nil
			}
			return cmdutil.RequiredArgs("database")(cmd, args)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if flags.interval <= 0 {
				return errors.New("--interval must be greater than 0")
			}
			if flags.duration < 0 {
				return errors.New("--duration must not be negative")
			}
			if flags.replay != "" && flags.capture != "" {
				return errors.New("--replay cannot be combined with --capture")
			}
			if flags.replay != "" && flags.duration > 0 {
				return errors.New("--duration cannot be combined with --replay")
			}
			if err := validateConnectionFilter(flags.instance, flags.role); err != nil {
				return err
			}
			if flags.replay != "" {
				if _, err := os.Stat(flags.replay); err != nil {
					return fmt.Errorf("--replay: %w", err)
				}
				if !isHumanMode(ch) {
					return errors.New("--replay requires an interactive terminal")
				}
			}
			if flags.capture != "" {
				if err := validateCapturePath(flags.capture); err != nil {
					return err
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTop(cmd.Context(), cmd, ch, args, flags)
		},
	}

	cmd.Flags().DurationVar(&flags.interval, "interval", 1*time.Second, "Refresh interval.")
	cmd.Flags().StringVar(&flags.capture, "capture", "", "Write captured samples to a trace file. Required in headless mode.")
	cmd.Flags().StringVar(&flags.replay, "replay", "", "Replay a previously captured trace file in the TUI. Mutually exclusive with --capture.")
	cmd.Flags().DurationVar(&flags.duration, "duration", 0, "Run for this duration. Default is to run until interrupted.")
	cmd.Flags().StringVar(&flags.instance, "instance", "", "Filter the live view to a single instance (by id from the list response).")
	cmd.Flags().StringVar(&flags.role, "role", "", "Filter the live view to rows whose instance role is primary or replica.")
	cmd.Flags().StringVar(&flags.keyspace, "keyspace", "", "Vitess keyspace to target.")
	cmd.Flags().StringVar(&flags.shard, "shard", "", "Vitess shard to target.")

	return cmd
}

type topFlags struct {
	interval time.Duration
	capture  string
	replay   string
	duration time.Duration
	instance string
	role     string
	keyspace string
	shard    string
}

func (f topFlags) filter() connectionFilter {
	return connectionFilter{instance: f.instance, role: f.role}
}

func (f topFlags) target() ConnectionTarget {
	return ConnectionTarget{Keyspace: f.keyspace, Shard: f.shard}
}

type topRequest struct {
	Database    string
	Branch      string
	Engine      ps.DatabaseEngine
	Filter      connectionFilter
	Target      ConnectionTarget
	Interactive bool
}

func runTop(ctx context.Context, cmd *cobra.Command, ch *cmdutil.Helper, args []string, flags topFlags) (err error) {
	if flags.replay != "" {
		return runReplay(ctx, flags.replay, flags.duration, flags.interval)
	}

	request, err := newTopRequest(ctx, ch, args, flags)
	if err != nil {
		return err
	}

	source, err := newTopSource(ctx, ch, request)
	if err != nil {
		return err
	}

	return runTopWithSource(ctx, cmd, ch, request, source, flags)
}

func newTopRequest(ctx context.Context, ch *cmdutil.Helper, args []string, flags topFlags) (topRequest, error) {
	database := args[0]
	engine, err := getTopDatabaseKind(ctx, ch, database)
	if err != nil {
		return topRequest{}, err
	}

	filter := flags.filter()
	target := flags.target()
	if err := validateTopFlagsForEngine(engine, filter, target); err != nil {
		return topRequest{}, err
	}

	branch, err := resolveBranch(ctx, ch, database, args)
	if err != nil {
		return topRequest{}, err
	}

	interactive := isHumanMode(ch)
	if !interactive && flags.capture == "" {
		return topRequest{}, errors.New("--capture is required when running without a TTY")
	}

	return topRequest{
		Database:    database,
		Branch:      branch,
		Engine:      engine,
		Filter:      filter,
		Target:      target,
		Interactive: interactive,
	}, nil
}

func runTopWithSource(ctx context.Context, cmd *cobra.Command, ch *cmdutil.Helper, request topRequest, source topSource, flags topFlags) error {
	if request.Filter.active() {
		fmt.Fprintln(cmd.ErrOrStderr(), request.Filter.describe())
	}

	if request.Interactive {
		return runTopInteractive(ctx, ch, request, source, flags)
	}
	return runTopHeadless(ctx, cmd, ch, request, source, flags)
}

func runTopInteractive(ctx context.Context, ch *cmdutil.Helper, request topRequest, source topSource, flags topFlags) error {
	control := newCaptureControl(flags.capture, ch.Config.Organization, request.Database, request.Branch, request.Filter, source.Target)
	if flags.capture != "" {
		writer, path, err := control.Open()
		if err != nil {
			return err
		}
		control.Writer = writer
		control.Path = path
	}
	target := tui.Target{
		Database: request.Database,
		Branch:   request.Branch,
		Keyspace: source.Target.Keyspace,
		Shard:    source.Target.Shard,
	}
	return runInteractive(ctx, source.Client, flags.duration, flags.interval, control, target, request.Filter.chip(), source.View)
}

func runTopHeadless(ctx context.Context, cmd *cobra.Command, ch *cmdutil.Helper, request topRequest, source topSource, flags topFlags) error {
	writer, err := openCaptureWriter(flags.capture, ch.Config.Organization, request.Database, request.Branch, request.Filter, source.Target)
	if err != nil {
		return err
	}

	if flags.duration > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "Capturing for %s to %s\n", flags.duration, flags.capture)
	} else {
		fmt.Fprintf(cmd.ErrOrStderr(), "Capturing to %s (Ctrl-C to stop)\n", flags.capture)
	}

	return runHeadlessCapture(ctx, sortedTopLister{client: source.Client, sort: source.View.DefaultSort()}, writer, flags.duration, flags.interval)
}

func getTopDatabaseKind(ctx context.Context, ch *cmdutil.Helper, database string) (ps.DatabaseEngine, error) {
	client, err := ch.Client()
	if err != nil {
		return "", err
	}
	db, err := client.Databases.Get(ctx, &ps.GetDatabaseRequest{
		Organization: ch.Config.Organization,
		Database:     database,
	})
	if err != nil {
		return "", err
	}
	if db == nil {
		return "", errors.New("database not found")
	}
	return db.Kind, nil
}

func validateTopFlagsForEngine(engine ps.DatabaseEngine, filter connectionFilter, target ConnectionTarget) error {
	if err := validateEngineFlags(engine, filter, target); err != nil {
		return err
	}
	switch engine {
	case ps.DatabaseEnginePostgres, ps.DatabaseEngineMySQL:
		return nil
	default:
		return fmt.Errorf("connections top is not supported for database kind %q", engine)
	}
}

func runReplay(ctx context.Context, path string, duration, interval time.Duration) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("--replay: %w", err)
	}
	defer file.Close()

	source, err := history.NewReplaySource(file)
	if err != nil {
		return fmt.Errorf("--replay: %w", err)
	}

	captures := source.Captures()
	samples := history.NewCaptureHistory(len(captures))
	for _, capture := range captures {
		samples.Push(capture.List)
	}
	view := replayConnectionView(captures)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	model := tui.NewModel(runCtx, newReplayClient(source), interval, duration).
		WithTarget(replayTarget(source)).
		WithConnectionView(view).
		WithCaptureHistory(samples).
		WithReadOnlyActions(errReplayActionRejected.Error())
	return runTeaProgram(model, tea.WithAltScreen(), tea.WithContext(runCtx))
}

func replayTarget(source *history.ReplaySource) tui.Target {
	start, ok := source.CaptureStart()
	if !ok {
		return tui.Target{}
	}
	target := tui.Target{
		Database: start.Database,
		Branch:   start.Branch,
	}
	if start.Target != nil {
		target.Keyspace = start.Target.Keyspace
		target.Shard = start.Target.Shard
	}
	return target
}

func replayConnectionView(captures []history.Capture) tui.ConnectionViewProfile {
	for _, capture := range captures {
		if capture.List.DatabaseKind == live.DatabaseKindMySQL {
			return tui.VitessConnectionView
		}
	}
	return tui.PostgresConnectionView
}

func isHumanMode(ch *cmdutil.Helper) bool {
	if !printer.IsTTY {
		return false
	}
	if ch.Printer != nil && ch.Printer.Format() != printer.Human {
		return false
	}
	return true
}

func resolveBranch(ctx context.Context, ch *cmdutil.Helper, database string, args []string) (string, error) {
	if len(args) >= 2 && args[1] != "" {
		return args[1], nil
	}
	client, err := ch.Client()
	if err != nil {
		return "", err
	}
	return promptutil.GetBranch(ctx, client, ch.Config.Organization, database)
}

var runTeaProgram = func(model tea.Model, options ...tea.ProgramOption) error {
	_, err := tea.NewProgram(model, options...).Run()
	return err
}

func runInteractive(ctx context.Context, client tui.ConnectionsClient, duration, interval time.Duration, control *tui.CaptureControl, target tui.Target, filterChip string, view tui.ConnectionViewProfile) (err error) {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if control != nil {
		defer func() {
			err = errors.Join(err, control.Close())
		}()
	}

	model := tui.NewModel(runCtx, client, interval, duration).
		WithTarget(target).
		WithFilter(filterChip).
		WithConnectionView(view)
	if control != nil {
		model = model.WithCaptureControl(control)
	}
	return runTeaProgram(model, tea.WithAltScreen(), tea.WithContext(runCtx))
}

type topSource struct {
	Client tui.ConnectionsClient
	View   tui.ConnectionViewProfile
	Target ConnectionTarget
}

func newTopSource(ctx context.Context, ch *cmdutil.Helper, request topRequest) (topSource, error) {
	switch request.Engine {
	case ps.DatabaseEnginePostgres:
		client, err := newConnectionsClient(ch, request.Database, request.Branch, ConnectionTarget{})
		if err != nil {
			return topSource{}, err
		}
		return topSource{
			Client: filteredLister{client: client, filter: request.Filter},
			View:   tui.PostgresConnectionView,
		}, nil
	case ps.DatabaseEngineMySQL:
		return newVitessTopSource(ctx, ch, request)
	default:
		return topSource{}, fmt.Errorf("connections top is not supported for database kind %q", request.Engine)
	}
}

func newVitessTopSource(ctx context.Context, ch *cmdutil.Helper, request topRequest) (topSource, error) {
	resolved, err := probeAndResolveVitessTopTarget(ctx, ch, request.Database, request.Branch, request.Target, request.Interactive)
	if err != nil {
		return topSource{}, err
	}
	client, err := newConnectionsClient(ch, request.Database, request.Branch, resolved)
	if err != nil {
		return topSource{}, err
	}
	return topSource{
		Client: filteredLister{client: client},
		View:   tui.VitessConnectionView,
		Target: resolved,
	}, nil
}

const maxVitessTopTargetProbes = 3

func probeAndResolveVitessTopTarget(ctx context.Context, ch *cmdutil.Helper, database, branch string, target ConnectionTarget, canPrompt bool) (ConnectionTarget, error) {
	resolved := target
	for attempts := 0; attempts < maxVitessTopTargetProbes; attempts++ {
		err := probeVitessTopTarget(ctx, ch, database, branch, resolved)
		if err == nil {
			return resolved, nil
		}

		next, ok, resolveErr := nextVitessTopTarget(resolved, err, canPrompt)
		if resolveErr != nil {
			return ConnectionTarget{}, resolveErr
		}
		if !ok {
			if canPrompt {
				return resolved, nil
			}
			return ConnectionTarget{}, err
		}
		resolved = next
	}
	return ConnectionTarget{}, errors.New("could not resolve Vitess keyspace and shard")
}

func probeVitessTopTarget(ctx context.Context, ch *cmdutil.Helper, database, branch string, target ConnectionTarget) error {
	client, err := newConnectionsClient(ch, database, branch, target)
	if err != nil {
		return err
	}
	_, err = client.List(ctx, live.SortByDuration)
	return err
}

func nextVitessTopTarget(target ConnectionTarget, err error, canPrompt bool) (ConnectionTarget, bool, error) {
	httpErr, ok := badRequestWithAlternatives(err)
	if !ok || !canPrompt {
		return ConnectionTarget{}, false, nil
	}

	switch {
	case target.Keyspace == "" && len(httpErr.Available.Keyspaces) > 0:
		selected, err := selectTopTarget("Select a keyspace for connections top:", httpErr.Available.Keyspaces)
		if err != nil {
			return ConnectionTarget{}, false, err
		}
		target.Keyspace = selected
		return target, true, nil
	case target.Shard == "" && len(httpErr.Available.Shards) > 0:
		selected, err := selectTopTarget("Select a shard for connections top:", httpErr.Available.Shards)
		if err != nil {
			return ConnectionTarget{}, false, err
		}
		target.Shard = selected
		return target, true, nil
	default:
		return ConnectionTarget{}, false, nil
	}
}

func badRequestWithAlternatives(err error) (*live.HTTPError, bool) {
	var httpErr *live.HTTPError
	if !errors.As(err, &httpErr) {
		return nil, false
	}
	return httpErr, httpErr.StatusCode == http.StatusBadRequest
}

var selectTopTarget = func(message string, options []string) (string, error) {
	prompt := &survey.Select{
		Message: message,
		Options: options,
		VimMode: true,
	}
	var selected string
	err := survey.AskOne(prompt, &selected)
	return selected, err
}

type sortedTopLister struct {
	client tui.ConnectionsClient
	sort   live.SortMode
}

func (l sortedTopLister) List(ctx context.Context, _ live.SortMode) (live.ConnectionList, error) {
	return l.client.List(ctx, l.sort)
}

func newCaptureControl(path, org, database, branch string, filter connectionFilter, target ConnectionTarget) *tui.CaptureControl {
	return &tui.CaptureControl{
		Open: func() (*history.CaptureWriter, string, error) {
			capturePath := path
			if capturePath == "" {
				capturePath = defaultInteractiveCapturePath(time.Now())
			}
			writer, err := openCaptureWriter(capturePath, org, database, branch, filter, target)
			return writer, capturePath, err
		},
	}
}

type connectionFilter struct {
	instance string
	role     string
}

func (f connectionFilter) active() bool {
	return f.instance != "" || f.role != ""
}

func validateConnectionFilter(instance, role string) error {
	if instance != "" && role != "" {
		return errors.New("--role cannot be combined with --instance")
	}
	switch role {
	case "", "primary", "replica":
		return nil
	default:
		return errors.New("--role must be primary or replica")
	}
}

// captureFilter renders the filter for the capture file header. Returns nil
// when no filter is active so the header's "filter" field is omitted.
func (f connectionFilter) captureFilter() *history.CaptureFilter {
	if !f.active() {
		return nil
	}
	return &history.CaptureFilter{
		Instance: f.instance,
		Role:     f.role,
	}
}

// chip renders the compact header indicator shown in the interactive TUI so
// the operator can see the view is scoped. Empty when no filter is active.
func (f connectionFilter) chip() string {
	switch {
	case f.instance != "":
		return fmt.Sprintf("filter: instance=%s", f.instance)
	case f.role != "":
		return fmt.Sprintf("filter: role=%s", f.role)
	default:
		return ""
	}
}

func (f connectionFilter) describe() string {
	switch {
	case f.instance != "":
		return fmt.Sprintf("filtering to instance=%s", f.instance)
	case f.role != "":
		return fmt.Sprintf("filtering to role=%s", f.role)
	default:
		return ""
	}
}

func filterConnectionList(list live.ConnectionList, f connectionFilter) live.ConnectionList {
	if !f.active() {
		return list
	}
	kept := make([]live.Connection, 0, len(list.Connections))
	for _, conn := range list.Connections {
		if f.instance != "" && conn.Instance != f.instance {
			continue
		}
		if f.role != "" && conn.InstanceRole != f.role {
			continue
		}
		kept = append(kept, conn)
	}
	keptInstances := make([]live.InstanceMeta, 0, len(list.Instances))
	for _, inst := range list.Instances {
		if f.instance != "" && inst.ID != f.instance {
			continue
		}
		if f.role != "" && inst.Role != f.role {
			continue
		}
		keptInstances = append(keptInstances, inst)
	}
	out := list
	out.Connections = kept
	out.Instances = keptInstances
	return out
}

type filteredLister struct {
	client *live.Client
	filter connectionFilter
}

func (f filteredLister) List(ctx context.Context, sort live.SortMode) (live.ConnectionList, error) {
	list, err := f.client.List(ctx, sort)
	if err != nil {
		return list, err
	}
	if err := validateInstanceFilter(list, f.filter); err != nil {
		return list, err
	}
	return filterConnectionList(list, f.filter), nil
}

// Actions pass through to the wire client unchanged — the filter only scopes
// the read path so the operator sees a subset of rows; once they pick a row,
// the action targets that specific (instance, pid) on the real cluster.
func (f filteredLister) CancelQuery(ctx context.Context, target live.ActionTarget) error {
	return f.client.CancelQuery(ctx, target)
}

func (f filteredLister) TerminateTransaction(ctx context.Context, target live.ActionTarget) error {
	return f.client.TerminateTransaction(ctx, target)
}

func (f filteredLister) TerminateConnection(ctx context.Context, target live.ActionTarget) error {
	return f.client.TerminateConnection(ctx, target)
}

func openCaptureWriter(path, org, database, branch string, filter connectionFilter, target ConnectionTarget) (*history.CaptureWriter, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	writer := history.NewCaptureWriter(file)
	if err := writer.WriteCaptureStart(history.CaptureStart{
		At:           time.Now().UTC(),
		Organization: org,
		Database:     database,
		Branch:       branch,
		Filter:       filter.captureFilter(),
		Target:       captureTarget(target),
	}); err != nil {
		_ = writer.Close()
		return nil, err
	}
	return writer, nil
}

func captureTarget(target ConnectionTarget) *history.CaptureTarget {
	if target.Keyspace == "" && target.Shard == "" {
		return nil
	}
	return &history.CaptureTarget{
		Keyspace: target.Keyspace,
		Shard:    target.Shard,
	}
}

func defaultInteractiveCapturePath(now time.Time) string {
	return "connections-" + now.UTC().Format("20060102T150405.000000000Z") + ".jsonl"
}

func validateCapturePath(path string) error {
	parent := filepath.Dir(path)
	info, err := os.Stat(parent)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("capture parent directory " + parent + " does not exist")
		}
		return err
	}
	if !info.IsDir() {
		return errors.New("capture parent path " + parent + " is not a directory")
	}
	return nil
}
