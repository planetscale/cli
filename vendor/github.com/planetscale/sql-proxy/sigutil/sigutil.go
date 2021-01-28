package sigutil

import (
	"context"
	"os"
	"os/signal"
)

// WithSignal returns a copy of the parent context with the context cancel
// function adjusted to be called when one of the given signals is received.
func WithSignal(ctx context.Context, sig ...os.Signal) context.Context {
	c := make(chan os.Signal, 1)
	signal.Notify(c, sig...)
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		select {
		case <-ctx.Done():
		case <-c:
		}

		cancel()
		signal.Stop(c)
	}()

	return ctx
}
