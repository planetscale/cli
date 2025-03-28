package printer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/gocarina/gocsv"
	"github.com/lensesio/tableprinter"
	"github.com/mattn/go-isatty"
)

var IsTTY = (isatty.IsTerminal(os.Stdout.Fd()) && isatty.IsTerminal(os.Stderr.Fd()) && isatty.IsTerminal(os.Stdin.Fd())) ||
	(isatty.IsCygwinTerminal(os.Stdout.Fd()) && isatty.IsCygwinTerminal(os.Stderr.Fd()) && isatty.IsCygwinTerminal(os.Stdin.Fd()))

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
	humanOut    io.Writer
	resourceOut io.Writer

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
// human, out returns io.Discard, which means that any output will be
// discarded
func (p *Printer) out() io.Writer {
	if p.humanOut != nil {
		return p.humanOut
	}

	if *p.format == Human {
		return color.Output
	}

	return io.Discard // /dev/nullj
}

// PrintProgress starts a spinner with the relevant message. The returned
// function needs to be called in a defer or when it's decided to stop the
// spinner
func (p *Printer) PrintProgress(message string) func() {
	if !IsTTY {
		fmt.Fprintln(p.out(), message)
		return func() {}
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(p.out()))
	s.Suffix = fmt.Sprintf(" %s", message)

	_ = s.Color("bold", "green")
	s.Start()
	return func() {
		s.Stop()

		// NOTE(fatih) the spinner library doesn't clear the line properly,
		// hence remove it ourselves. This line should be removed once it's
		// fixed in upstream.  https://github.com/briandowns/spinner/pull/117
		fmt.Fprint(p.out(), "\r\033[2K")
	}
}

// Format returns the format that was set for this printer
func (p *Printer) Format() Format { return *p.format }

// SetHumanOutput sets the output for human readable messages.
func (p *Printer) SetHumanOutput(out io.Writer) {
	p.humanOut = out
}

// SetResourceOutput sets the output for pringing resources via PrintResource.
func (p *Printer) SetResourceOutput(out io.Writer) {
	p.resourceOut = out
}

// PrintResource prints the given resource in the format it was specified.
func (p *Printer) PrintResource(v interface{}) error {
	if p.format == nil {
		return errors.New("printer.Format is not set")
	}

	var out io.Writer = os.Stdout
	if p.resourceOut != nil {
		out = p.resourceOut
	}

	switch *p.format {
	case Human:
		var b strings.Builder
		tableprinter.Print(&b, v)
		fmt.Fprintln(out, b.String())
		return nil
	case JSON:
		return p.PrintJSON(v)
	case CSV:
		type csvvaluer interface {
			MarshalCSVValue() interface{}
		}

		if c, ok := v.(csvvaluer); ok {
			v = c.MarshalCSVValue()
		}

		buf, err := gocsv.MarshalString(v)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, buf)
		return nil
	}

	return fmt.Errorf("unknown printer.Format: %T", *p.format)
}

func (p *Printer) ConfirmCommand(confirmationName, commandShortName, confirmFailedName string) error {
	if p.Format() != Human {
		return fmt.Errorf("cannot %s with the output format %q (run with -force to override)", commandShortName, p.Format())
	}

	if !IsTTY {
		return fmt.Errorf("cannot confirm %s %q (run with -force to override)", confirmFailedName, confirmationName)
	}

	confirmationMessage := fmt.Sprintf("%s %s %s", Bold("Please type"), BoldBlue(confirmationName), Bold("to confirm:"))

	prompt := &survey.Input{
		Message: confirmationMessage,
	}

	var userInput string
	err := survey.AskOne(prompt, &userInput)
	if err != nil {
		if err == terminal.InterruptErr {
			os.Exit(0)
		} else {
			return err
		}
	}

	// If the confirmations don't match up, let's return an error.
	if userInput != confirmationName {
		return fmt.Errorf("incorrect value entered, skipping %s", commandShortName)
	}

	return nil
}

func (p *Printer) PrintJSON(v interface{}) error {
	var out io.Writer = os.Stdout
	if p.resourceOut != nil {
		out = p.resourceOut
	}

	buf, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	fmt.Fprintln(out, string(buf))
	return nil
}

func (p *Printer) PrettyPrintJSON(b []byte) error {
	var out io.Writer = os.Stdout
	if p.resourceOut != nil {
		out = p.resourceOut
	}

	var buf bytes.Buffer
	err := json.Indent(&buf, b, "", "  ")
	if err != nil {
		return err
	}

	fmt.Fprintln(out, buf.String())
	return nil
}

func GetMilliseconds(timestamp time.Time) int64 {
	if timestamp.IsZero() {
		return 0
	}

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
func BoldBlue(msg interface{}) string {
	// the 'color' package already handles IsTTY gracefully
	return color.New(color.FgBlue).Add(color.Bold).Sprint(msg)
}

// BoldRed returns a string formatted with red and bold.
func BoldRed(msg interface{}) string {
	return color.New(color.FgRed).Add(color.Bold).Sprint(msg)
}

// BoldGreen returns a string formatted with green and bold.
func BoldGreen(msg interface{}) string {
	return color.New(color.FgGreen).Add(color.Bold).Sprint(msg)
}

// BoldBlack returns a string formatted with Black and bold.
func BoldBlack(msg interface{}) string {
	return color.New(color.FgBlack).Add(color.Bold).Sprint(msg)
}

// BoldYellow returns a string formatted with yellow and bold.
func BoldYellow(msg interface{}) string {
	return color.New(color.FgYellow).Add(color.Bold).Sprint(msg)
}

// Red returns a string formatted with red and bold.
func Red(msg interface{}) string {
	return color.New(color.FgRed).Sprint(msg)
}

// Bold returns a string formatted with bold.
func Bold(msg interface{}) string {
	// the 'color' package already handles IsTTY gracefully
	return color.New(color.Bold).Sprint(msg)
}

// Number returns a formatted number with the # prefix
func Number(number uint64) string {
	return fmt.Sprintf("#%d", number)
}
