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
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	os.Exit(cmd.Execute(ctx, version, commit, date))
}
