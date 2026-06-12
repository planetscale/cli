package connections

import (
	"context"
	"errors"
	"strings"

	"github.com/planetscale/cli/internal/cmdutil"
	live "github.com/planetscale/cli/internal/connections"
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
)

type actionResult struct {
	Success  bool   `csv:"success" header:"success" json:"success"`
	Keyspace string `csv:"keyspace" header:"keyspace" json:"keyspace,omitempty"`
	Shard    string `csv:"shard" header:"shard" json:"shard,omitempty"`
	Tablet   string `csv:"tablet" header:"tablet" json:"tablet,omitempty"`
	ID       int64  `csv:"id" header:"id,text" json:"id,omitempty"`
	Kind     string `csv:"kind" header:"kind" json:"kind,omitempty"`
}

func (a *actionResult) MarshalCSVValue() interface{} {
	return []*actionResult{a}
}

type compactActionResult struct {
	Success bool   `csv:"success" header:"success" json:"success"`
	ID      int64  `csv:"id" header:"id,text" json:"id,omitempty"`
	Kind    string `csv:"kind" header:"kind" json:"kind,omitempty"`
}

func (a *compactActionResult) MarshalCSVValue() interface{} {
	return []*compactActionResult{a}
}

func toActionResult(result live.ActionResult) *actionResult {
	return &actionResult{
		Success:  result.Success,
		Keyspace: result.Keyspace,
		Shard:    result.Shard,
		Tablet:   result.Tablet,
		ID:       result.ID,
		Kind:     result.Kind,
	}
}

func toCompactActionResult(result live.ActionResult) *compactActionResult {
	return &compactActionResult{
		Success: result.Success,
		ID:      result.ID,
		Kind:    result.Kind,
	}
}

// RunCancelQuery cancels the active query identified by a live connection query ID.
func RunCancelQuery(ctx context.Context, ch *cmdutil.Helper, database, branch, queryID string, target ConnectionTarget) error {
	return RunCancelQueryForEngine(ctx, ch, database, branch, queryID, ps.DatabaseEngineMySQL, target)
}

// RunCancelQueryForEngine cancels the active query and prints output for the resolved database engine.
func RunCancelQueryForEngine(ctx context.Context, ch *cmdutil.Helper, database, branch, queryID string, engine ps.DatabaseEngine, target ConnectionTarget) error {
	return runAction(ctx, ch, database, branch, "query-id", queryID, target, func(ctx context.Context, client *live.Client, id string) (live.ActionResult, error) {
		return client.CancelQueryResult(ctx, live.ActionTarget{QueryID: &id})
	}, engine)
}

// RunKillTransaction terminates the connection identified by a live connection transaction ID.
func RunKillTransaction(ctx context.Context, ch *cmdutil.Helper, database, branch, transactionID string, target ConnectionTarget) error {
	return RunKillTransactionForEngine(ctx, ch, database, branch, transactionID, ps.DatabaseEnginePostgres, target)
}

// RunKillTransactionForEngine terminates a transaction and prints output for the resolved database engine.
func RunKillTransactionForEngine(ctx context.Context, ch *cmdutil.Helper, database, branch, transactionID string, engine ps.DatabaseEngine, target ConnectionTarget) error {
	return runAction(ctx, ch, database, branch, "transaction-id", transactionID, target, func(ctx context.Context, client *live.Client, id string) (live.ActionResult, error) {
		return client.TerminateTransactionResult(ctx, live.ActionTarget{TransactionID: &id})
	}, engine)
}

// RunKillConnection terminates the connection identified by a live connection_id.
func RunKillConnection(ctx context.Context, ch *cmdutil.Helper, database, branch, connectionID string, target ConnectionTarget) error {
	return RunKillConnectionForEngine(ctx, ch, database, branch, connectionID, ps.DatabaseEngineMySQL, target)
}

// RunKillConnectionForEngine terminates a connection and prints output for the resolved database engine.
func RunKillConnectionForEngine(ctx context.Context, ch *cmdutil.Helper, database, branch, connectionID string, engine ps.DatabaseEngine, target ConnectionTarget) error {
	return runAction(ctx, ch, database, branch, "connection-id", connectionID, target, func(ctx context.Context, client *live.Client, id string) (live.ActionResult, error) {
		return client.TerminateConnectionResult(ctx, live.ActionTarget{ConnectionID: &id})
	}, engine)
}

func runAction(ctx context.Context, ch *cmdutil.Helper, database, branch, idName, id string, target ConnectionTarget, runAction func(context.Context, *live.Client, string) (live.ActionResult, error), engine ps.DatabaseEngine) error {
	if err := validateActionID(idName, id); err != nil {
		return err
	}
	id = strings.TrimSpace(id)

	client, err := newConnectionsClient(ch, database, branch, target)
	if err != nil {
		return err
	}

	result, err := runAction(ctx, client, id)
	if err != nil {
		return err
	}
	return printActionResult(ch, result, engine, idName)
}

// ValidateConnectionID checks the connection action identifier without making network calls.
func ValidateConnectionID(id string) error {
	return validateActionID("connection-id", id)
}

// ValidateQueryID checks the query action identifier without making network calls.
func ValidateQueryID(id string) error {
	return validateActionID("query-id", id)
}

// ValidateTransactionID checks the transaction action identifier without making network calls.
func ValidateTransactionID(id string) error {
	return validateActionID("transaction-id", id)
}

func validateActionID(idName, id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New(idName + " is required")
	}
	return nil
}

func printActionResult(ch *cmdutil.Helper, result live.ActionResult, engine ps.DatabaseEngine, idName string) error {
	if ch.Printer.Format() == printer.Human {
		ch.Printer.Printf("%s.\n", actionResultMessage(result, idName))
		return nil
	}
	if ch.Printer.Format() == printer.JSON {
		return ch.Printer.PrintResource(toActionResult(result))
	}
	if engine == ps.DatabaseEnginePostgres {
		return ch.Printer.PrintResource(toCompactActionResult(result))
	}
	return ch.Printer.PrintResource(toActionResult(result))
}

func actionResultMessage(result live.ActionResult, idName string) string {
	var message string
	switch idName {
	case "query-id":
		message = "Cancelled query"
	case "transaction-id":
		message = "Killed transaction"
	case "connection-id":
		message = "Killed connection"
	default:
		message = "Action sent"
	}
	if result.Tablet != "" {
		message += " on " + result.Tablet
	}
	return message
}
