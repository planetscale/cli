package oauthapplication

import (
	"encoding/json"

	"github.com/planetscale/cli/internal/cmdutil"
	ps "github.com/planetscale/cli/internal/planetscale"
	"github.com/planetscale/cli/internal/printer"
	"github.com/spf13/cobra"
)

// OAuthApplicationCmd encapsulates the command for managing OAuth applications.
func OAuthApplicationCmd(ch *cmdutil.Helper) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "oauth-application <command>",
		Short:             "List and manage OAuth applications and their tokens",
		PersistentPreRunE: cmdutil.CheckAuthentication(ch.Config),
	}

	cmd.PersistentFlags().StringVar(&ch.Config.Organization, "org", ch.Config.Organization, "The organization for the current user")
	cmd.MarkPersistentFlagRequired("org") // nolint:errcheck

	cmd.AddCommand(ListCmd(ch))
	cmd.AddCommand(ShowCmd(ch))
	cmd.AddCommand(TokenCmd(ch))
	return cmd
}

// OAuthApplication returns a table and JSON serializable OAuth application.
type OAuthApplication struct {
	ID          string `header:"id" json:"id"`
	Name        string `header:"name" json:"name"`
	ClientID    string `header:"client_id" json:"client_id"`
	RedirectURI string `header:"redirect_uri" json:"redirect_uri"`
	Tokens      int    `header:"tokens" json:"tokens"`
	CreatedAt   int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	UpdatedAt   int64  `header:"updated_at,timestamp(ms|utc|human)" json:"updated_at"`

	orig *ps.OAuthApplication
}

func (o *OAuthApplication) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(o.orig, "", "  ")
}

func toOAuthApplication(application *ps.OAuthApplication) *OAuthApplication {
	return &OAuthApplication{
		ID:          application.ID,
		Name:        application.Name,
		ClientID:    application.ClientID,
		RedirectURI: application.RedirectURI,
		Tokens:      application.Tokens,
		CreatedAt:   printer.GetMilliseconds(application.CreatedAt),
		UpdatedAt:   printer.GetMilliseconds(application.UpdatedAt),
		orig:        application,
	}
}

func toOAuthApplications(applications []*ps.OAuthApplication) []*OAuthApplication {
	results := make([]*OAuthApplication, 0, len(applications))
	for _, application := range applications {
		results = append(results, toOAuthApplication(application))
	}
	return results
}

// OAuthToken returns a table and JSON serializable OAuth token.
type OAuthToken struct {
	ID               string `header:"id" json:"id"`
	DisplayName      string `header:"display_name" json:"display_name"`
	ActorDisplayName string `header:"actor_display_name" json:"actor_display_name"`
	LastUsedAt       int64  `header:"last_used_at,timestamp(ms|utc|human)" json:"last_used_at"`
	ExpiresAt        int64  `header:"expires_at,timestamp(ms|utc|human)" json:"expires_at"`
	CreatedAt        int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`

	orig *ps.OAuthToken
}

func (o *OAuthToken) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(o.orig, "", "  ")
}

func toOAuthToken(token *ps.OAuthToken) *OAuthToken {
	var actorDisplayName string
	if token.ActorDisplayName != nil {
		actorDisplayName = *token.ActorDisplayName
	}

	var lastUsedAt int64
	if token.LastUsedAt != nil {
		lastUsedAt = printer.GetMilliseconds(*token.LastUsedAt)
	}

	var expiresAt int64
	if token.ExpiresAt != nil {
		expiresAt = printer.GetMilliseconds(*token.ExpiresAt)
	}

	return &OAuthToken{
		ID:               token.ID,
		DisplayName:      token.DisplayName,
		ActorDisplayName: actorDisplayName,
		LastUsedAt:       lastUsedAt,
		ExpiresAt:        expiresAt,
		CreatedAt:        printer.GetMilliseconds(token.CreatedAt),
		orig:             token,
	}
}

func toOAuthTokens(tokens []*ps.OAuthToken) []*OAuthToken {
	results := make([]*OAuthToken, 0, len(tokens))
	for _, token := range tokens {
		results = append(results, toOAuthToken(token))
	}
	return results
}

// OAuthTokenWithSecret includes the plaintext credentials returned after creation.
type OAuthTokenWithSecret struct {
	ID                    string `header:"id" json:"id"`
	DisplayName           string `header:"display_name" json:"display_name"`
	ActorDisplayName      string `header:"actor_display_name" json:"actor_display_name"`
	LastUsedAt            int64  `header:"last_used_at,timestamp(ms|utc|human)" json:"last_used_at"`
	ExpiresAt             int64  `header:"expires_at,timestamp(ms|utc|human)" json:"expires_at"`
	CreatedAt             int64  `header:"created_at,timestamp(ms|utc|human)" json:"created_at"`
	Token                 string `header:"token" json:"token"`
	PlainTextRefreshToken string `header:"plain_text_refresh_token" json:"plain_text_refresh_token"`

	orig *ps.OAuthToken
}

func (o *OAuthTokenWithSecret) MarshalJSON() ([]byte, error) {
	return json.MarshalIndent(o.orig, "", "  ")
}

func toOAuthTokenWithSecret(token *ps.OAuthToken) *OAuthTokenWithSecret {
	plainToken, refreshToken := "", ""
	if token.Token != nil {
		plainToken = *token.Token
	}
	if token.PlainTextRefreshToken != nil {
		refreshToken = *token.PlainTextRefreshToken
	}
	result := toOAuthToken(token)
	return &OAuthTokenWithSecret{
		ID:                    result.ID,
		DisplayName:           result.DisplayName,
		ActorDisplayName:      result.ActorDisplayName,
		LastUsedAt:            result.LastUsedAt,
		ExpiresAt:             result.ExpiresAt,
		CreatedAt:             result.CreatedAt,
		Token:                 plainToken,
		PlainTextRefreshToken: refreshToken,
		orig:                  token,
	}
}
