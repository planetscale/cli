package d1

import (
	"strings"

	"github.com/planetscale/cli/internal/printer"
)

const completeReminderShort = "ORM migration tables were not imported. Re-baseline your migration history on Postgres before cutover."

// CompleteResult is the data payload for import d1 complete.
type CompleteResult struct {
	MigrationID      string   `json:"migration_id"`
	Status           string   `json:"status"`
	SkippedORMTables []string `json:"skipped_orm_tables,omitempty"`
}

// CompleteResponse builds the success envelope for import d1 complete.
func CompleteResponse(org, database, branch, migrationID string) (Response, error) {
	state, err := LoadState(org, database, branch, migrationID)
	if err != nil {
		return Response{}, err
	}

	skippedTables, nextSteps, err := completeORMNextSteps(state.InputPath)
	if err != nil {
		return Response{}, err
	}

	resp := OKResponse("complete", CompleteResult{
		MigrationID:      migrationID,
		Status:           PhaseComplete,
		SkippedORMTables: skippedTables,
	}, nextSteps)
	resp.MigrationID = migrationID
	resp.Phase = PhaseComplete
	resp.Reminder = completeReminderShort
	return resp, nil
}

// CompleteSlackMessage returns a short Slack-friendly completion line.
func CompleteSlackMessage(skippedTables []string, orms []string) string {
	if len(skippedTables) == 0 {
		return "D1 import marked complete. If your app uses an ORM, re-baseline migration history on Postgres before cutover."
	}
	msg := "Data import complete. Next: re-baseline ORM migrations on Postgres — migration history tables from D1 were not imported."
	if len(orms) > 0 {
		msg += " Detected: " + strings.Join(orms, ", ") + "."
	}
	return msg
}

func completeORMNextSteps(inputPath string) (skippedTables []string, steps []NextStep, err error) {
	if inputPath == "" {
		return nil, genericORMCompleteNextSteps(), nil
	}

	tables, err := ParseDump(inputPath)
	if err != nil {
		return nil, nil, err
	}

	seenORM := make(map[string]struct{})
	for _, table := range tables {
		rule := ORMMetadataRule(table.Name)
		if rule == nil {
			continue
		}
		skippedTables = append(skippedTables, table.Name)
		if _, ok := seenORM[rule.orm]; ok {
			continue
		}
		seenORM[rule.orm] = struct{}{}
		steps = append(steps, NextStep{
			Tool:   rule.orm,
			Reason: rule.remediation,
		})
	}

	if len(steps) == 0 {
		steps = genericORMCompleteNextSteps()
	}
	return skippedTables, steps, nil
}

func genericORMCompleteNextSteps() []NextStep {
	return []NextStep{{
		Tool: "ORM migrations",
		Reason: "If your app uses an ORM or migration framework (Drizzle, Prisma, Rails, etc.), " +
			"re-baseline migration history on Postgres. SQLite bookkeeping tables are never imported.",
	}}
}

func ormNamesFromSkippedTables(skippedTables []string) []string {
	seen := make(map[string]struct{})
	var names []string
	for _, table := range skippedTables {
		rule := ORMMetadataRule(table)
		if rule == nil {
			continue
		}
		if _, ok := seen[rule.orm]; ok {
			continue
		}
		seen[rule.orm] = struct{}{}
		names = append(names, rule.orm)
	}
	return names
}

func printCompleteReminderHuman(p *printer.Printer, result CompleteResult) {
	p.Println("\nReminder: ORM migration history was not imported")
	p.Println("  Application data is in Postgres, but framework tables such as")
	p.Println("  __drizzle_migrations, _prisma_migrations, and schema_migrations")
	p.Println("  were intentionally skipped.")
	if len(result.SkippedORMTables) > 0 {
		p.Printf("  Skipped in this export: %s\n", strings.Join(result.SkippedORMTables, ", "))
	}
	p.Println("\n  Before cutover:")
	p.Println("  • Point your app at the PlanetScale Postgres branch")
	p.Println("  • Re-baseline migrations on Postgres (do not copy SQLite history)")
	p.Println("  • Run your ORM's mark-applied / fake-initial flow for the current schema")
	p.Println("\n  Run pscale import d1 lint --input <export> for ORM-specific guidance.")
}
