package d1

import (
	"strings"
	"testing"
)

func TestCompleteResponseIncludesORMGuidance(t *testing.T) {
	t.Setenv("PSCALE_TEST_MODE", "1")

	org, database, branch := "acme", "mydb", "main"
	migrationID := "complete-response-123"
	if err := SavePlan(&PlanResult{
		MigrationID: migrationID,
		Org:         org,
		Database:    database,
		Branch:      branch,
		InputPath:   testFixture(t),
	}); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	if err := SetMigrationPhase(org, database, branch, migrationID, PhaseVerified); err != nil {
		t.Fatalf("SetMigrationPhase: %v", err)
	}

	resp, err := CompleteResponse(org, database, branch, migrationID)
	if err != nil {
		t.Fatalf("CompleteResponse: %v", err)
	}
	if resp.Reminder == "" {
		t.Fatal("expected reminder")
	}
	if len(resp.NextSteps) == 0 {
		t.Fatal("expected ORM next steps")
	}

	data, ok := resp.Data.(CompleteResult)
	if !ok {
		t.Fatalf("data type = %T, want CompleteResult", resp.Data)
	}
	if len(data.SkippedORMTables) == 0 {
		t.Fatal("expected skipped ORM tables from sample fixture")
	}

	foundDrizzle := false
	foundPrisma := false
	for _, step := range resp.NextSteps {
		switch step.Tool {
		case "Drizzle":
			foundDrizzle = true
		case "Prisma":
			foundPrisma = true
		}
	}
	if !foundDrizzle || !foundPrisma {
		t.Fatalf("next steps = %#v, want Drizzle and Prisma", resp.NextSteps)
	}
}

func TestCompleteSlackMessageWithDetectedORMs(t *testing.T) {
	msg := CompleteSlackMessage(
		[]string{"__drizzle_migrations", "_prisma_migrations"},
		[]string{"Drizzle", "Prisma"},
	)
	if !strings.Contains(msg, "re-baseline ORM migrations") {
		t.Fatalf("message = %q", msg)
	}
	if !strings.Contains(msg, "Drizzle, Prisma") {
		t.Fatalf("message = %q", msg)
	}
}

func TestCompleteSlackMessageWithoutORMTables(t *testing.T) {
	msg := CompleteSlackMessage(nil, nil)
	if !strings.Contains(msg, "re-baseline migration history") {
		t.Fatalf("message = %q", msg)
	}
}
