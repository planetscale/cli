package cmdutil

import (
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

// PrintProgress starts a spinner with the relevant message.
func PrintProgress(message string) func() {
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
	return color.New(color.FgBlue).Add(color.Bold).Sprint(msg)
}

// Bold returns a string formatted with bold.
func Bold(msg string) string {
	return color.New(color.Bold).Sprint(msg)
}
