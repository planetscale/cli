package cmdutil

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/planetscale/cli/internal/config"
	"github.com/planetscale/planetscale-go/planetscale"
	"github.com/spf13/cobra"
)

// ValidateServiceTokenOnNotFound checks if a 404 error might be due to invalid service token authentication.
// It performs an additional authentication check when using service tokens to distinguish between
// a genuine "not found" error and an authentication failure.
//
// This function should be called when:
// 1. The API returns a 404 error
// 2. The user is authenticated with a service token
//
// Returns:
// - nil if the service token is valid (indicating a genuine 404)
// - An authentication error if the service token is invalid
// - The original error if not using service tokens or if the validation check fails
func ValidateServiceTokenOnNotFound(ctx context.Context, cmd *cobra.Command, cfg *config.Config, clientFunc func() (*planetscale.Client, error), originalErr error) error {
	// Only perform this check if we're using service token authentication
	if !cfg.ServiceTokenIsSet() {
		return originalErr
	}

	// Try to make a simple API call to validate the service token
	client, err := clientFunc()
	if err != nil {
		// If we can't create a client, return the original error
		return originalErr
	}

	// Try to list organizations as a simple authentication check
	// This is the same check used in auth.CheckCmd
	_, err = client.Organizations.List(ctx)
	if err != nil {
		// If this call fails, it's likely an authentication issue
		return &Error{
			Msg:      buildServiceTokenErrorMessage(cmd),
			ExitCode: ActionRequestedExitCode,
		}
	}

	// If the auth check passes, the original 404 is genuine
	return originalErr
}

// buildServiceTokenErrorMessage creates a helpful error message based on how the user is authenticating
func buildServiceTokenErrorMessage(cmd *cobra.Command) string {
	// Check if service token values were set via environment variables
	envTokenID := os.Getenv("PLANETSCALE_SERVICE_TOKEN_ID")
	envToken := os.Getenv("PLANETSCALE_SERVICE_TOKEN")

	// Check if flags were explicitly set via command line
	var flagsUsed bool
	if cmd != nil {
		flagsUsed = cmd.Flags().Changed("service-token-id") || cmd.Flags().Changed("service-token")
	}

	var authMethod string
	var specificGuidance string

	if envTokenID != "" || envToken != "" {
		authMethod = "environment variables"
		specificGuidance = "Please check your PLANETSCALE_SERVICE_TOKEN_ID and PLANETSCALE_SERVICE_TOKEN environment variables."
	} else if flagsUsed {
		authMethod = "command-line flags"
		specificGuidance = "Please check your --service-token-id and --service-token flag values."
	} else {
		// Fallback - we know service tokens are set but can't determine the source
		authMethod = "service tokens"
		specificGuidance = "Please check your service token credentials."
	}

	return fmt.Sprintf(`Authentication failed. Your service token appears to be invalid.

You are currently authenticating using %s. %s

Service tokens can be provided in two ways:
  1. Environment variables: PLANETSCALE_SERVICE_TOKEN_ID and PLANETSCALE_SERVICE_TOKEN
  2. Command-line flags: --service-token-id and --service-token

To create a new service token, run: pscale service-token create
To list existing service tokens, run: pscale service-token list`, authMethod, specificGuidance)
}

// HandleNotFoundWithServiceTokenCheck is a convenience function that combines the common pattern
// of handling 404 errors with service token validation. It should be used in the ErrNotFound case
// of error handling switch statements.
//
// The requiredPermission parameter is optional (can be empty string). When provided, it will be
// included in the error message to help users understand what permission their service token needs.
//
// Usage example:
//
//	switch cmdutil.ErrCode(err) {
//	case planetscale.ErrNotFound:
//	    return cmdutil.HandleNotFoundWithServiceTokenCheck(
//	        ctx, cmd, ch.Config, ch.Client, err, "read_branch",
//	        "branch %s does not exist in database %s (organization: %s)",
//	        printer.BoldBlue(branch), printer.BoldBlue(database), printer.BoldBlue(ch.Config.Organization))
//	default:
//	    return cmdutil.HandleError(err)
//	}
func HandleNotFoundWithServiceTokenCheck(ctx context.Context, cmd *cobra.Command, cfg *config.Config, clientFunc func() (*planetscale.Client, error), originalErr error, requiredPermission string, notFoundMsgFormat string, args ...interface{}) error {
	// First check if this might be a service token authentication issue
	if authErr := ValidateServiceTokenOnNotFound(ctx, cmd, cfg, clientFunc, originalErr); authErr != originalErr {
		return authErr
	}

	// If authentication is valid, return the formatted not found message
	baseMsg := fmt.Sprintf(notFoundMsgFormat, args...)

	// If using service tokens and a required permission is specified, add permission guidance
	if cfg.ServiceTokenIsSet() && requiredPermission != "" {
		return fmt.Errorf("%s\n\nNote: You are using a service token for authentication. If this resource exists, your service token may not have the required '%s' permission to access it. Please check your service token permissions.", baseMsg, requiredPermission)
	}

	return errors.New(baseMsg)
}
