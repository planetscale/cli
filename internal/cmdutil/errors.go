package cmdutil

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/planetscale/planetscale-go/planetscale"
)

const ActionRequestedExitCode = 1
const FatalErrExitCode = 2

var errExpiredAuthMessage = errors.New("the access token has expired. Please run 'pscale auth login'")

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
		// TODO(fatih): fix the return type in our API.
		// authErrorResponse represents an error response from the API
		type authErrorResponse struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
			State            string `json:"state"`
		}

		errorRes := &authErrorResponse{}
		mErr := json.Unmarshal([]byte(perr.Meta["body"]), errorRes)
		if mErr != nil {
			// return back original error (not *mErr*). Looks like the error is
			// not an authentication error
			return fmt.Errorf("%s with the following output:\n\n%s", perr.Error(), perr.Meta["body"])
		}

		if errorRes.Error == "invalid_token" {
			return errExpiredAuthMessage
		}

		return err
	default:
		return err
	}
}
