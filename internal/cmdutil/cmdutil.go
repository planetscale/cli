package cmdutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/cli/internal/printer"

	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	exec "golang.org/x/sys/execabs"
)

// Helper is passed to every single command and is used by individual
// subcommands.
type Helper struct {
	// Config contains globally sourced configuration
	Config *config.Config

	ConfigFS *config.ConfigFS

	// Client returns the PlanetScale API client
	Client func() (*ps.Client, error)

	// Printer is used to print output of a command to stdout.
	Printer *printer.Printer

	// bebug defines the debug mode
	debug *bool
}

func (h *Helper) SetDebug(debug *bool) {
	h.debug = debug
}

func (h *Helper) Debug() bool { return *h.debug }

// required arguments are not available.
func RequiredArgs(reqArgs ...string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		n := len(reqArgs)
		if len(args) >= n {
			return nil
		}

		missing := reqArgs[len(args):]

		a := fmt.Sprintf("arguments <%s>", strings.Join(missing, ", "))
		if len(missing) == 1 {
			a = fmt.Sprintf("argument <%s>", missing[0])
		}

		return fmt.Errorf("missing %s \n\n%s", a, cmd.UsageString())
	}
}

// CheckAuthentication checks whether the user is authenticated and returns a
// actionable error message.
func CheckAuthentication(cfg *config.Config) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if cfg.IsAuthenticated() {
			return nil
		}

		return errors.New("not authenticated yet. Please run 'pscale auth login'")
	}
}

// NewZapLogger returns a logger to be used with the sql-proxy. By default it
// only outputs error leveled messages, unless debug is true.
func NewZapLogger(debug bool) *zap.Logger {
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		NameKey:        "logger",
		TimeKey:        "T",
		EncodeLevel:    zapcore.LowercaseColorLevelEncoder,
		EncodeTime:     zapcore.RFC3339TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}

	level := zap.ErrorLevel
	if debug {
		level = zap.DebugLevel
	}

	logger := zap.New(zapcore.NewCore(zapcore.NewConsoleEncoder(encoderCfg), os.Stdout, level))

	return logger
}

// IsUnderHomebrew checks whether the given binary is under the homebrew path.
// copied from: https://github.com/cli/cli/blob/trunk/cmd/gh/main.go#L298
func IsUnderHomebrew(binpath string) bool {
	if binpath == "" {
		return false
	}

	brewExe, err := exec.LookPath("brew")
	if err != nil {
		return false
	}

	brewPrefixBytes, err := exec.Command(brewExe, "--prefix").Output()
	if err != nil {
		return false
	}

	brewBinPrefix := filepath.Join(strings.TrimSpace(string(brewPrefixBytes)), "bin") + string(filepath.Separator)
	return strings.HasPrefix(binpath, brewBinPrefix)
}

// HasHomebrew check whether the user has installed brew
func HasHomebrew() bool {
	_, err := exec.LookPath("brew")
	if err == nil {
		return true
	}
	return false
}

// MySQLClientPath checks whether the 'mysql' client exists and returns the
// path to the binary. The returned error contains instructions to install the
// client.
func MySQLClientPath() (string, error) {
	path, err := exec.LookPath("mysql")
	if err == nil {
		return path, nil
	}

	msg := "couldn't find the 'msyql' client required to run this command."
	installURL := "https://dev.mysql.com/doc/mysql-shell/8.0/en/mysql-shell-install.html"

	switch runtime.GOOS {
	case "darwin":
		if HasHomebrew() {
			return "", fmt.Errorf("%s\nTo install, run: brew install mysql-client", msg)
		}

		installURL = "https://dev.mysql.com/doc/mysql-shell/8.0/en/mysql-shell-install-macos-quick.html"
	case "linux":
		installURL = "https://dev.mysql.com/doc/mysql-shell/8.0/en/mysql-shell-install-linux-quick.html"
	case "windows":
		installURL = "https://dev.mysql.com/doc/mysql-shell/8.0/en/mysql-shell-install-windows-quick.html"
	}

	return "", fmt.Errorf("%s\nTo install, follow the instructions: %s", msg, installURL)
}
