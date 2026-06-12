package tui

import live "github.com/planetscale/cli/internal/connections"

type actionTargetField int

const (
	actionTargetUnset actionTargetField = iota
	actionTargetQueryID
	actionTargetTransactionID
	actionTargetConnectionID
)

// ActionRequirement describes the row identifier required before an action can
// be offered for a selected connection. A zero requirement means the action is
// not available for the connection source.
type ActionRequirement struct {
	field actionTargetField
}

var (
	// ActionTargetQueryID requires a query_id before the action can run.
	ActionTargetQueryID = ActionRequirement{field: actionTargetQueryID}
	// ActionTargetTransactionID requires a transaction_id before the action can run.
	ActionTargetTransactionID = ActionRequirement{field: actionTargetTransactionID}
	// ActionTargetConnectionID requires a connection_id before the action can run.
	ActionTargetConnectionID = ActionRequirement{field: actionTargetConnectionID}
)

// ConnectionCapabilities is the TUI capability contract for a connection source.
// It tells the view which selected-row IDs are needed for each action, whether
// blocker navigation should be shown, and which action controls should be
// visible.
//
// It does not execute actions; the model still calls ConnectionsClient. The
// support value only gates visible controls and validates that the selected row
// has the identifier the backend expects.
type ConnectionCapabilities struct {
	CancelQuery          ActionRequirement
	TerminateTransaction ActionRequirement
	TerminateConnection  ActionRequirement
	ShowBlockers         bool

	configured bool
}

// DefaultConnectionCapabilities is the Postgres capability set. A zero
// ConnectionCapabilities also resolves to this default so existing tests and
// model construction keep Postgres behavior unless a source opts into different
// capabilities.
func DefaultConnectionCapabilities() ConnectionCapabilities {
	return ConnectionCapabilities{
		CancelQuery:          ActionTargetQueryID,
		TerminateTransaction: ActionTargetTransactionID,
		TerminateConnection:  ActionTargetConnectionID,
		ShowBlockers:         true,
		configured:           true,
	}
}

func (s ConnectionCapabilities) effective() ConnectionCapabilities {
	if !s.configured && s.isZero() {
		return DefaultConnectionCapabilities()
	}
	return s
}

func (s ConnectionCapabilities) isZero() bool {
	return s.CancelQuery == (ActionRequirement{}) &&
		s.TerminateTransaction == (ActionRequirement{}) &&
		s.TerminateConnection == (ActionRequirement{}) &&
		!s.ShowBlockers
}

func (s ConnectionCapabilities) supports(kind actionKind) bool {
	requirement := s.requirement(kind)
	return requirement.field != actionTargetUnset
}

func (s ConnectionCapabilities) requirement(kind actionKind) ActionRequirement {
	s = s.effective()
	switch kind {
	case actionCancelQuery:
		return s.CancelQuery
	case actionTerminateTxn:
		return s.TerminateTransaction
	case actionTerminateConn:
		return s.TerminateConnection
	default:
		return ActionRequirement{}
	}
}

func (s ConnectionCapabilities) missingActionID(kind actionKind, target live.ActionTarget) string {
	requirement := s.requirement(kind)
	switch requirement.field {
	case actionTargetQueryID:
		if live.DerefString(target.QueryID) == "" {
			return "no active query to cancel on this connection"
		}
	case actionTargetTransactionID:
		if live.DerefString(target.TransactionID) == "" {
			return "no open transaction to terminate on this connection"
		}
	case actionTargetConnectionID:
		if live.DerefString(target.ConnectionID) == "" {
			return "no connection id available to terminate this connection"
		}
	default:
		return "action is not supported for this connection"
	}
	return ""
}
