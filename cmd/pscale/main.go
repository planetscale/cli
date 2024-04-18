package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/planetscale/cli/internal/cmd"
)

var (
	version string
	commit  string
	date    string
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigc := make(chan os.Signal, 1)
	signals := []os.Signal{os.Interrupt}
	signal.Notify(sigc, signals...)
	defer func() {
		signal.Stop(sigc)
		cancel()
	}()
	go func() {
		select {
		case <-sigc:
			cancel()
		case <-ctx.Done():
		}
	}()

	return cmd.Execute(ctx, sigc, signals, version, commit, date)
}
