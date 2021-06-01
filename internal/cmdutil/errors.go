package cmdutil

import (
	"fmt"

	"github.com/planetscale/planetscale-go/planetscale"
)

// Error can be used by a command to change the exit status of the CLI.
type Error struct {
	Msg string
	// Status
	ExitCode int
}

func (e *Error) Error() string { return e.Msg }

// ErrCode returns the code from a *planetscale.Error, if available. If the
// error is not of type *planetscale.Error or is nil, it returns an empty,
// undefined error code.
func ErrCode(err error) planetscale.ErrorCode {
	if err == nil {
		return ""
	}

	perr, ok := err.(*planetscale.Error)
	if !ok {
		return ""
	}

	return perr.Code

}

// HandleError checks whether the given err is an *planetscale.Error and
// returns a descriptive, human readable error if the error code matches a
// certain planetscale Error types. If the error doesn't match these
// requirements, err is returned unmodified.
func HandleError(err error) error {
	if err == nil {
		return err
	}

	perr, ok := err.(*planetscale.Error)
	if !ok {
		return err
	}

	switch perr.Code {
	case planetscale.ErrResponseMalformed:
		const malformedWarning = "Unexpected API response received, the PlanetScale API might be down." +
			" Please contact support with the following output"

		return fmt.Errorf("%s:\n\n%s", malformedWarning, perr.Meta["body"])
	case planetscale.ErrInternal:
		return fmt.Errorf("%s with the following output:\n\n%s", perr.Error(), perr.Meta["body"])
	default:
		return err
	}
}
