package cmdutil

import (
	"fmt"

	"github.com/planetscale/planetscale-go/planetscale"
)

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

// MalformedError checks whether the given err is an *planetscale.Error and
// returns a descriptive, human readable error if the error code is of type
// planetscale.ErrResponseMalformed. If the error doesn't match these
// requirements, err is returned unmodified.
func MalformedError(err error) error {
	if err == nil {
		return err
	}

	perr, ok := err.(*planetscale.Error)
	if !ok {
		return err
	}

	if perr.Code != planetscale.ErrResponseMalformed {
		return err
	}

	const malformedWarning = "Unexpected API response received, the PlanetScale API might be down." +
		" Please contact support with the following output"

	return fmt.Errorf("%s:\n\n%s", malformedWarning, perr.Meta["body"])
}
