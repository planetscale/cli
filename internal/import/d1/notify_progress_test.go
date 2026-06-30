package d1

import "testing"

func TestShouldNotifyProgressMajorStages(t *testing.T) {
	for _, stage := range []string{
		ImportStageConnecting,
		ImportStageSQLiteStaging,
		ImportStageSchema,
		ImportStageIndexes,
		ImportStageSequences,
	} {
		if !shouldNotifyProgress(ImportProgress{Stage: stage}) {
			t.Fatalf("expected stage %q to notify", stage)
		}
	}
}

func TestShouldNotifyProgressPgloaderTables(t *testing.T) {
	for _, current := range []int{1, 2, 19} {
		if !shouldNotifyProgress(ImportProgress{Stage: ImportStagePgloader, Current: current, Total: 19, Detail: "users"}) {
			t.Fatalf("expected pgloader table %d to notify", current)
		}
	}
}

func TestShouldNotifyProgressRowCounts(t *testing.T) {
	for _, current := range []int{1, 2, 19} {
		if !shouldNotifyProgress(ImportProgress{Stage: VerifyStageRowCounts, Current: current, Total: 19, Detail: "users"}) {
			t.Fatalf("expected row count %d to notify", current)
		}
	}
}

func TestFormatProgressMessageSQLiteStaging(t *testing.T) {
	got := FormatProgressMessage(ImportProgress{Stage: ImportStageSQLiteStaging})
	want := "Staging SQLite database from export..."
	if got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
}

func TestShouldNotifyProgressUnknownStage(t *testing.T) {
	if shouldNotifyProgress(ImportProgress{Stage: "custom_stage", Current: 1, Detail: "working"}) {
		t.Fatal("expected unknown stage to skip slack notification")
	}
}
