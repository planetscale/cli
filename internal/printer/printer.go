package printer

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	ps "github.com/planetscale/planetscale-go/planetscale"
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
// human, out returns ioutil.Discard, which means that any output will be
// discarded
func (p *Printer) out() io.Writer {
	if p.humanOut != nil {
		return p.humanOut
	}

	if *p.format == Human {
		return color.Output
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

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(p.out()))
	s.Suffix = fmt.Sprintf(" %s", message)

	s.Color("bold", "green") // nolint:errcheck
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
		buf, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}

		fmt.Fprintln(out, string(buf))
		return nil
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
		return errors.New(fmt.Sprintf("incorrect value entered, skipping %s", commandShortName))
	}

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

func (p *Printer) PrintDataImport(di ps.DataImport) {
	completedSteps := GetCompletedImportStates(di.ImportState)
	if len(completedSteps) > 0 {
		p.Println(completedSteps)
	}

	inProgressStep, _ := GetCurrentImportState(di.ImportState)
	if len(inProgressStep) > 0 {
		p.Println(inProgressStep)
	}

	pendingSteps := GetPendingImportStates(di.ImportState)
	if len(pendingSteps) > 0 {
		p.Println(pendingSteps)
	}
}
func GetCompletedImportStates(state ps.DataImportState) string {
	completedStates := []string{}
	switch state {
	case ps.DataImportCopyingData:
		completedStates = append(completedStates, BoldGreen("1. Started Data Copy"))
	case ps.DataImportSwitchTrafficPending, ps.DataImportSwitchTrafficError:
		completedStates = append(completedStates, BoldGreen("1. Started Data Copy"))
		completedStates = append(completedStates, BoldGreen("2. Copied Data"))
	case ps.DataImportSwitchTrafficCompleted:
		completedStates = append(completedStates, BoldGreen("1. Started Data Copy"))
		completedStates = append(completedStates, BoldGreen("2. Copied Data"))
		completedStates = append(completedStates, BoldGreen("3. Running as Replica"))

	case ps.DataImportReady:
		completedStates = append(completedStates, BoldGreen("1. Started Data Copy"))
		completedStates = append(completedStates, BoldGreen("2. Copied Data"))
		completedStates = append(completedStates, BoldGreen("3. Running as replica"))
		completedStates = append(completedStates, BoldGreen("4. Running as Primary"))
	}
	return strings.Join(completedStates, "\n")
}

func GetCurrentImportState(d ps.DataImportState) (string, bool) {
	switch d {
	case ps.DataImportPreparingDataCopy:
		return BoldYellow("> 1. Starting Data Copy"), true
	case ps.DataImportPreparingDataCopyFailed:
		return BoldRed("> 1. Cannot Start Data Copy"), false
	case ps.DataImportCopyingData:
		return BoldYellow("> 2. Copying Data"), true
	case ps.DataImportCopyingDataFailed:
		return BoldRed("> 2. Failed to Copy Data"), false
	case ps.DataImportSwitchTrafficPending:
		return BoldYellow("> 3. Running as Replica"), true
	case ps.DataImportSwitchTrafficRunning:
		return BoldYellow("> 3. Switching to primary"), true
	case ps.DataImportSwitchTrafficError:
		return BoldRed("> 3. Failed switching to primary"), false
	case ps.DataImportReverseTrafficRunning:
		return BoldYellow("3. Switching to replica"), true
	case ps.DataImportSwitchTrafficCompleted:
		return BoldYellow("> 4. Running as Primary"), true
	case ps.DataImportReverseTrafficError:
		return BoldRed("> 3. Failed switching to primary"), false
	case ps.DataImportDetachExternalDatabaseRunning:
		return BoldYellow("> 4. detaching external database"), true
	case ps.DataImportDetachExternalDatabaseError:
		return BoldRed("> 4. failed to detach external database"), false
	case ps.DataImportReady:
		return BoldGreen("> 5. Ready"), false
	}

	panic("unhandled state " + d.String())
}

func GetPendingImportStates(state ps.DataImportState) string {
	var pendingStates []string
	switch state {
	case ps.DataImportPreparingDataCopy:
		pendingStates = append(pendingStates, BoldBlack("2. Copied Data"))
		pendingStates = append(pendingStates, BoldBlack("3. Running as Replica"))
		pendingStates = append(pendingStates, BoldBlack("4. Running as Primary"))
		pendingStates = append(pendingStates, BoldBlack("5. Detached external database"))
	case ps.DataImportCopyingData:
		pendingStates = append(pendingStates, BoldBlack("3. Running as Replica"))
		pendingStates = append(pendingStates, BoldBlack("4. Running as Primary"))
		pendingStates = append(pendingStates, BoldBlack("5. Detached external database"))
	case ps.DataImportSwitchTrafficPending:
		pendingStates = append(pendingStates, BoldBlack("4. Running as Primary"))
		pendingStates = append(pendingStates, BoldBlack("5. Detached external database"))
	case ps.DataImportSwitchTrafficCompleted:
		pendingStates = append(pendingStates, BoldBlack("5. Detached external database"))
	}
	return strings.Join(pendingStates, "\n")
}

func ImportProgress(state ps.DataImportState) string {
	preparingDataCopyState := color.New(color.FgBlack).Add(color.Bold).Sprint("1. Started Data Copy")
	dataCopyState := color.New(color.FgBlack).Add(color.Bold).Sprint("2. Copied Data")
	switchingTrafficState := color.New(color.FgBlack).Add(color.Bold).Sprint("3. Running as Replica")
	detachExternalDatabaseState := color.New(color.FgBlack).Add(color.Bold).Sprint("4. Running as Primary")
	readyState := color.New(color.FgBlack).Add(color.Bold).Sprint("5. Detached External Database, ready to use")
	// Preparing data copy > Data copying > Running as Replica > Running as Primary > Detach External database
	switch state {
	case ps.DataImportPreparingDataCopy:
		preparingDataCopyState = color.New(color.FgYellow).Add(color.Bold).Sprint(state.String())
		break
	case ps.DataImportSwitchTrafficPending:
		preparingDataCopyState = color.New(color.FgGreen).Add(color.Bold).Sprint("1. Started Data Copy")
		dataCopyState = color.New(color.FgGreen).Add(color.Bold).Sprint("2. Copied Data")
		switchingTrafficState = color.New(color.FgYellow).Add(color.Bold).Sprint("3. Running as Replica")
		break
	case ps.DataImportPreparingDataCopyFailed:
		preparingDataCopyState = color.New(color.FgGreen).Add(color.Bold).Sprint("1. Cannot start data copy")
		break
	}

	return strings.Join([]string{preparingDataCopyState, dataCopyState, switchingTrafficState, detachExternalDatabaseState, readyState}, "\n")

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
