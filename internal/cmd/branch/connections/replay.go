package connections

import (
	"context"
	"errors"

	live "github.com/planetscale/cli/internal/connections"
	"github.com/planetscale/cli/internal/connections/history"
)

// replayClient adapts a *history.ReplaySource to the tui.ConnectionsClient
// interface so the TUI can render captured snapshots. Action methods all
// reject with the same operator-visible message so c/k/K surface as errors in
// the footer rather than being dispatched to the real wire.
type replayClient struct {
	source *history.ReplaySource
}

func newReplayClient(source *history.ReplaySource) *replayClient {
	return &replayClient{source: source}
}

func (r *replayClient) List(ctx context.Context, mode live.SortMode) (live.ConnectionList, error) {
	return r.source.List(ctx, mode)
}

func (r *replayClient) CancelQuery(context.Context, live.ActionTarget) error {
	return errReplayActionRejected
}

func (r *replayClient) TerminateTransaction(context.Context, live.ActionTarget) error {
	return errReplayActionRejected
}

func (r *replayClient) TerminateConnection(context.Context, live.ActionTarget) error {
	return errReplayActionRejected
}

var errReplayActionRejected = errors.New("not available in replay mode")
