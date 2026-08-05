package printer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
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

func (f Format) String() string {
	switch f {
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
	handle := p.StartProgress(message)
	return handle.Stop
}

// ProgressHandle is an updatable progress indicator (spinner on TTY).
type ProgressHandle struct {
	update func(string)
	stop   func()
}

// Update changes the progress message.
func (h *ProgressHandle) Update(message string) {
	if h != nil && h.update != nil {
		h.update(message)
	}
}

// Stop ends the progress indicator.
func (h *ProgressHandle) Stop() {
	if h != nil && h.stop != nil {
		h.stop()
	}
}

// StartProgress starts a spinner or line-based progress on w when not a TTY.
func (p *Printer) StartProgress(message string) *ProgressHandle {
	return p.startProgressOn(p.out(), message)
}

func (p *Printer) startProgressOn(w io.Writer, message string) *ProgressHandle {
	if !IsTTY {
		fmt.Fprintln(w, message)
		return &ProgressHandle{
			update: func(msg string) { fmt.Fprintln(w, msg) },
		}
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(w))
	s.Suffix = fmt.Sprintf(" %s", message)

	_ = s.Color("bold", "green")
	s.Start()
	return &ProgressHandle{
		update: func(msg string) { s.Suffix = fmt.Sprintf(" %s", msg) },
		stop: func() {
			s.Stop()
			// NOTE(fatih) the spinner library doesn't clear the line properly,
			// hence remove it ourselves. This line should be removed once it's
			// fixed in upstream.  https://github.com/briandowns/spinner/pull/117
			fmt.Fprint(w, "\r\033[2K")
		},
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

// ResourceOutput returns the writer PrintResource writes to, for commands that
// render resources themselves (e.g. dynamic column sets that can't use struct
// tags).
func (p *Printer) ResourceOutput() io.Writer {
	if p.resourceOut != nil {
		return p.resourceOut
	}
	return os.Stdout
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
		tableprinter.Print(&b, tableValue(v))
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
		return fmt.Errorf(`cannot %s with the output format "%s" (run with --force to override)`, commandShortName, p.Format())
	}

	if !IsTTY {
		return fmt.Errorf("cannot confirm %s %q (run with --force to override)", confirmFailedName, confirmationName)
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

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}

	fmt.Fprint(out, buf.String())
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

// tableValue prepares a resource for tableprinter. A nil pointer field yields
// no cell at all, which slides every following value one column to the left and
// under the wrong header, so pointers to scalars are replaced by their element
// type with nil becoming the zero value. Zero renders as the ",n/a" header tag
// value (or blank for timestamps), and JSON and CSV output are unaffected.
func tableValue(v interface{}) interface{} {
	rv := reflect.ValueOf(v)

	switch rv.Kind() {
	case reflect.Pointer:
		if rv.IsNil() || rv.Elem().Kind() != reflect.Struct {
			return v
		}
		return flattenStruct(rv.Elem()).Interface()
	case reflect.Struct:
		return flattenStruct(rv).Interface()
	case reflect.Slice, reflect.Array:
		elem := rv.Type().Elem()
		if elem.Kind() == reflect.Pointer {
			elem = elem.Elem()
		}
		if elem.Kind() != reflect.Struct {
			return v
		}

		flat := flatStructType(elem)
		if flat == elem {
			return v
		}

		out := reflect.MakeSlice(reflect.SliceOf(flat), 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			row := rv.Index(i)
			if row.Kind() == reflect.Pointer {
				if row.IsNil() {
					out = reflect.Append(out, reflect.New(flat).Elem())
					continue
				}
				row = row.Elem()
			}
			out = reflect.Append(out, flattenStruct(row))
		}
		return out.Interface()
	}

	return v
}

func flattenStruct(rv reflect.Value) reflect.Value {
	flat := flatStructType(rv.Type())
	if flat == rv.Type() {
		return rv
	}

	out := reflect.New(flat).Elem()
	for i := 0; i < flat.NumField(); i++ {
		field := flat.Field(i)
		from := rv.FieldByName(field.Name)

		if from.Type() == field.Type {
			out.Field(i).Set(from)
			continue
		}
		if !from.IsNil() {
			out.Field(i).Set(from.Elem())
		}
	}

	return out
}

var (
	flatStructTypesMu sync.RWMutex
	flatStructTypes   = map[reflect.Type]reflect.Type{}
)

// flatStructType returns typ with every pointer-to-scalar field replaced by the
// type it points to, or typ itself when there is nothing to replace.
func flatStructType(typ reflect.Type) reflect.Type {
	flatStructTypesMu.RLock()
	cached, ok := flatStructTypes[typ]
	flatStructTypesMu.RUnlock()
	if ok {
		return cached
	}

	flat := typ
	fields := make([]reflect.StructField, 0, typ.NumField())
	hasPointer := false

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// reflect.StructOf cannot rebuild these, and tableprinter ignores
		// unexported fields anyway.
		if field.PkgPath != "" {
			continue
		}
		if field.Anonymous {
			hasPointer = false
			break
		}

		if field.Type.Kind() == reflect.Pointer && isScalarKind(field.Type.Elem().Kind()) {
			field.Type = field.Type.Elem()
			hasPointer = true
		}
		fields = append(fields, field)
	}

	if hasPointer {
		flat = reflect.StructOf(fields)
	}

	flatStructTypesMu.Lock()
	flatStructTypes[typ] = flat
	flatStructTypesMu.Unlock()

	return flat
}

func isScalarKind(k reflect.Kind) bool {
	switch k {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
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
