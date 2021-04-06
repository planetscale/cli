package cmdutil

import (
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/planetscale/planetscale-go/planetscale"
)

var IsTTY = isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())

func Emoji(emoji string) string {
	if IsTTY {
		return emoji
	}
	return ""
}

// PrintProgress starts a spinner with the relevant message.
func PrintProgress(message string) func() {
	if !IsTTY {
		fmt.Println(message)
		return func() {}
	}

	// Output to STDERR so we don't polluate STDOUT.
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
	s.Suffix = fmt.Sprintf(" %s", message)

	s.Color("bold", "green") // nolint:errcheck
	s.Start()
	return func() {
		s.Stop()
	}
}

// BoldBlue returns a string formatted with blue and bold.
func BoldBlue(msg string) string {
	// the 'color' package already handles IsTTY gracefully
	return color.New(color.FgBlue).Add(color.Bold).Sprint(msg)
}

// Bold returns a string formatted with bold.
func Bold(msg string) string {
	// the 'color' package already handles IsTTY gracefully
	return color.New(color.Bold).Sprint(msg)
}

func IsNotFoundError(err error) bool {
	if pErr, ok := err.(*planetscale.Error); ok {
		if pErr.Code == planetscale.ErrNotFound {
			return true
		}
	}

	return false
}
