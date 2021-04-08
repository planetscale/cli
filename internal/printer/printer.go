package printer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/gocarina/gocsv"
	"github.com/lensesio/tableprinter"
	"github.com/mattn/go-isatty"
)

var IsTTY = isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())

// Format defines the option output format of a resource.
type Format int

const (
	// Human prints it in human readable format. This can be either a table or
	// a single line, depending on the resource implementation.
	Human Format = iota
	JSON
	CSV
)

// NewFormatValue is used to define a flag that can be used to define a custom
// flag via the flagset.Var() method.
func NewFormatValue(val Format, p *Format) *Format {
	*p = val
	return (*Format)(p)
}

func (f *Format) String() string {
	switch *f {
	case Human:
		return "human"
	case JSON:
		return "json"
	case CSV:
		return "csv"
	}

	return "unknown format"
}

func (f *Format) Set(s string) error {
	var v Format
	switch s {
	case "human":
		v = Human
	case "json":
		v = JSON
	case "csv":
		v = CSV
	default:
		return fmt.Errorf("failed to parse Format: %q. Valid values: %+v",
			s, []string{"human", "json", "csv"})
	}

	*f = Format(v)
	return nil
}

func (f *Format) Type() string {
	return "string"
}

// Printer is used to print information to the defined output.
type Printer struct {
	format *Format
}

// NewPrinter returns a new Printer for the given output and format.
func NewPrinter(format *Format) *Printer {
	return &Printer{
		format: format,
	}
}

// Printf is a convenience method to Printf to the defined output.
func (p *Printer) Printf(format string, i ...interface{}) {
	fmt.Fprintf(p.out(), format, i...)
}

// Println is a convenience method to Println to the defined output.
func (p *Printer) Println(i ...interface{}) {
	fmt.Fprintln(p.out(), i...)
}

// Print is a convenience method to Print to the defined output.
func (p *Printer) Print(i ...interface{}) {
	fmt.Fprint(p.out(), i...)
}

// out defines the output to write human readable text. If format is not set to
// human, out returns ioutil.Discard, which means that any output will be
// discarded
func (p *Printer) out() io.Writer {
	if *p.format == Human {
		return os.Stdout
	}

	return ioutil.Discard // /dev/nullj
}

// PrintProgress starts a spinner with the relevant message. The returned
// function needs to be called in a defer or when it's decided to stop the
// spinner
func (p *Printer) PrintProgress(message string) func() {
	if !IsTTY {
		fmt.Fprintln(p.out(), message)
		return func() {}
	}

	// Output to STDERR so we don't polluate STDOUT.
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(p.out()))
	s.Suffix = fmt.Sprintf(" %s", message)

	s.Color("bold", "green") // nolint:errcheck
	s.Start()
	return func() {
		s.Stop()
	}
}

// Format returns the format that was set for this printer
func (p *Printer) Format() Format { return *p.format }

// PrintResource prints the given resource in the format it was specified.
func (p *Printer) PrintResource(v interface{}) error {
	if p.format == nil {
		return errors.New("printer.Format is not set")
	}

	switch *p.format {
	case Human:
		var b strings.Builder
		tableprinter.Print(&b, v)
		fmt.Println(b.String())
		return nil
	case JSON:
		out, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}

		fmt.Print(string(out))
		return nil
	case CSV:
		out, err := gocsv.MarshalString(v)
		if err != nil {
			return err
		}
		fmt.Print(out)

		return nil
	}

	return fmt.Errorf("unknown printer.Format: %T", *p.format)
}

func GetMilliseconds(timestamp time.Time) int64 {
	numSeconds := timestamp.UTC().UnixNano() /
		(int64(time.Millisecond) / int64(time.Nanosecond))

	return numSeconds
}

func GetMillisecondsIfExists(timestamp *time.Time) *int64 {
	if timestamp == nil {
		return nil
	}

	numSeconds := GetMilliseconds(*timestamp)

	return &numSeconds
}

func Emoji(emoji string) string {
	if IsTTY {
		return emoji
	}
	return ""
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
