package connections

import (
	"context"
	"errors"
	"time"

	live "github.com/planetscale/cli/internal/connections"
	"github.com/planetscale/cli/internal/connections/history"
)

// runHeadlessCapture writes captures through writer until duration elapses or
// ctx is canceled. A non-positive duration runs continuously until ctx fires.
func runHeadlessCapture(ctx context.Context, client clientInterface, writer captureWriter, duration, interval time.Duration) (err error) {
	defer func() {
		err = errors.Join(err, writer.Close())
	}()

	if duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, duration)
		defer cancel()
	}

	captured := false
	finalize := func() error {
		if duration <= 0 {
			return nil
		}
		if captured {
			return nil
		}
		return errors.New("capture produced no samples")
	}

	list, err := client.List(ctx, live.SortByTransactionStart)
	if err != nil {
		if ctx.Err() != nil {
			return finalize()
		}
		return err
	}
	if err := writer.Write(history.NewCapture(list)); err != nil {
		return err
	}
	captured = true

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return finalize()
		case <-ticker.C:
			list, err := client.List(ctx, live.SortByTransactionStart)
			if err != nil {
				if ctx.Err() != nil {
					return finalize()
				}
				return err
			}
			if err := writer.Write(history.NewCapture(list)); err != nil {
				return err
			}
			captured = true
		}
	}
}

type clientInterface interface {
	List(context.Context, live.SortMode) (live.ConnectionList, error)
}

type captureWriter interface {
	Write(history.Capture) error
	Close() error
}
