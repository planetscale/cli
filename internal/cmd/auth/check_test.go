package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/config"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

func TestCheckCmdDoesNotShadowAPIURLFlag(t *testing.T) {
	cmd := CheckCmd(&cmdutil.Helper{})
	if cmd.Flags().Lookup("api-url") != nil {
		t.Fatal("auth check must not define a local --api-url flag")
	}
}

func TestCheckCmdUsesRootPersistentAPIURL(t *testing.T) {
	const customURL = "https://api.custom.example/v1"

	format := printer.JSON
	var out bytes.Buffer
	cfg := &config.Config{BaseURL: ps.DefaultBaseURL}
	ch := &cmdutil.Helper{
		Printer: printer.NewPrinter(&format),
		Config:  cfg,
		Client: func() (*ps.Client, error) {
			return cfg.NewClientFromConfig()
		},
	}
	ch.Printer.SetResourceOutput(&out)

	root := &cobra.Command{Use: "pscale", SilenceErrors: true, SilenceUsage: true}
	root.PersistentFlags().StringVar(&cfg.BaseURL, "api-url", ps.DefaultBaseURL, "The base URL for the PlanetScale API.")
	root.PersistentFlags().VarP(printer.NewFormatValue(printer.Human, &format), "format", "f", "output format")
	root.AddCommand(AuthCmd(ch))
	root.SetArgs([]string{"--api-url", customURL, "--format", "json", "auth", "check"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected exit error for unauthenticated auth check")
	}
	var cmdErr *cmdutil.Error
	if !errors.As(err, &cmdErr) || !cmdErr.Handled {
		t.Fatalf("expected handled JSON exit error, got %v", err)
	}

	var resp AuthCheckResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("stdout json: %v, out=%q", err, out.String())
	}
	if resp.APIURL != customURL {
		t.Fatalf("api_url = %q, want %q", resp.APIURL, customURL)
	}
}
