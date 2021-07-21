package auditlog

import (
	"fmt"
	"net/url"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var auditLogEvents = map[string]ps.AuditLogEvent{
	"branch.created":                  ps.AuditLogEventBranchCreated,
	"branch.deleted":                  ps.AuditLogEventBranchDeleted,
	"database.created":                ps.AuditLogEventDatabaseCreated,
	"database.deleted":                ps.AuditLogEventDatabaseDeleted,
	"deploy_request.approved":         ps.AuditLogEventDeployRequestApproved,
	"deploy_request.closed":           ps.AuditLogEventDeployRequestClosed,
	"deploy_request.created":          ps.AuditLogEventDeployRequestCreated,
	"deploy_request.deleted":          ps.AuditLogEventDeployRequestDeleted,
	"deploy_request.queued":           ps.AuditLogEventDeployRequestQueued,
	"deploy_request.unqueued":         ps.AuditLogEventDeployRequestUnqueued,
	"integration.created":             ps.AuditLogEventIntegrationCreated,
	"integration.deleted":             ps.AuditLogEventIntegrationDeleted,
	"organization_invitation.created": ps.AuditLogEventOrganizationInvitationCreated,
	"organization_invitation.deleted": ps.AuditLogEventOrganizationInvitationDeleted,
	"organization_membership.created": ps.AuditLogEventOrganizationMembershipCreated,
	"organization.joined":             ps.AuditLogEventOrganizationJoined,
	"organization.removed_member":     ps.AuditLogEventOrganizationRemovedMember,
	"organization.disabled_sso":       ps.AuditLogEventOrganizationDisabledSSO,
	"organization.enabled_sso":        ps.AuditLogEventOrganizationEnabledSSO,
	"organization.updated_role":       ps.AuditLogEventOrganizationUpdatedRole,
	"service_token.created":           ps.AuditLogEventServiceTokenCreated,
	"service_token.deleted":           ps.AuditLogEventServiceTokenDeleted,
	"service_token.granted_access":    ps.AuditLogEventServiceTokenGrantedAccess,
}

// ListCmd encapsulates the command for listing audit logs for an organization.
func ListCmd(ch *cmdutil.Helper) *cobra.Command {
	var flags struct {
		action []string
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all audit logs of an organization",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			web, err := cmd.Flags().GetBool("web")
			if err != nil {
				return err
			}

			if web {
				fmt.Println("🌐  Redirecting you to your audit logs in your web browser.")
				path := fmt.Sprintf("%s/%s/settings/audit-log", cmdutil.ApplicationURL, ch.Config.Organization)

				v := url.Values{}
				if len(flags.action) != 0 {
					for _, action := range flags.action {
						v.Add("filters[]", fmt.Sprintf("audit_action:%s", action))
					}
				}

				if vals := v.Encode(); vals != "" {
					path += "?" + vals
				}

				err := browser.OpenURL(path)
				if err != nil {
					return err
				}
				return nil
			}

			client, err := ch.Client()
			if err != nil {
				return err
			}

			end := ch.Printer.PrintProgress(fmt.Sprintf("Fetching audit logs for %s", printer.BoldBlue(ch.Config.Organization)))
			defer end()

			auditLogs, err := client.AuditLogs.List(ctx, &ps.ListAuditLogsRequest{
				Organization: ch.Config.Organization,
				Events:       toAuditLogEvents(flags.action),
			})
			if err != nil {
				switch cmdutil.ErrCode(err) {
				case ps.ErrNotFound:
					return fmt.Errorf("audit logs does not exist in organization: %s (are you an 'admin'?)",
						printer.BoldBlue(ch.Config.Organization))
				default:
					return cmdutil.HandleError(err)
				}
			}
			end()

			if len(auditLogs) == 0 && ch.Printer.Format() == printer.Human {
				ch.Printer.Printf("No audit logs exist for organization %s.\n",
					printer.BoldBlue(ch.Config.Organization))
				return nil
			}

			return ch.Printer.PrintResource(toAuditLogs(auditLogs))
		},
	}

	cmd.Flags().BoolP("web", "w", false, "List audit logs in your web browser.")
	cmd.Flags().StringSliceVar(&flags.action, "action", nil, "Filter based on the action type")
	cmd.RegisterFlagCompletionFunc("action", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		actions := make([]string, 0, len(auditLogEvents))
		for action := range auditLogEvents {
			actions = append(actions, action)
		}

		return actions, cobra.ShellCompDirectiveDefault
	})
	return cmd
}

func toAuditLogEvents(actions []string) []ps.AuditLogEvent {
	var events []ps.AuditLogEvent

	for _, action := range actions {
		events = append(events, auditLogEvents[action])
	}

	return events
}
