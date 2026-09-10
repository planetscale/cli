package cmdutil

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestExactArgs(t *testing.T) {
	cmd := &cobra.Command{Use: "test <database> <branch>"}
	validate := ExactArgs("database", "branch")

	if err := validate(cmd, []string{"app", "main"}); err != nil {
		t.Fatalf("exact arguments returned error: %v", err)
	}

	if err := validate(cmd, []string{"app"}); err == nil || !strings.Contains(err.Error(), "missing argument <branch>") {
		t.Fatalf("missing argument error = %v", err)
	}

	if err := validate(cmd, []string{"app", "main", "extra"}); err == nil || !strings.Contains(err.Error(), "accepts 2 arg(s), received 3") {
		t.Fatalf("extra argument error = %v", err)
	}
}
