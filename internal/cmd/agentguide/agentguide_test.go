package agentguide

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/printer"
)

func TestAgentGuideJSONBootstrap(t *testing.T) {
	format := printer.JSON
	var out bytes.Buffer
	ch := &cmdutil.Helper{
		Printer: printer.NewPrinter(&format),
	}
	ch.Printer.SetResourceOutput(&out)

	cmd := AgentGuideCmd(ch)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var resp response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q", resp.Status)
	}
	if resp.FirstCommand != cmdutil.AgentAuthCheckCmd() {
		t.Fatalf("first command = %q", resp.FirstCommand)
	}
	if resp.HostedMCPURL != HostedMCPURL {
		t.Fatalf("hosted MCP URL = %q", resp.HostedMCPURL)
	}
	if resp.SkillsRepoURL != SkillsRepoURL {
		t.Fatalf("skills repo URL = %q", resp.SkillsRepoURL)
	}
	if resp.SkillsSetupCommand != SkillsSetupCmd {
		t.Fatalf("skills setup command = %q", resp.SkillsSetupCommand)
	}
	if resp.SkillsCLIAutomation != SkillsCLIAutomation {
		t.Fatalf("skills CLI automation skill = %q", resp.SkillsCLIAutomation)
	}
	if len(resp.NextSteps) < 2 || resp.NextSteps[1] != SkillsSetupCmd {
		t.Fatalf("next_steps = %#v", resp.NextSteps)
	}
	if resp.Guide == "" {
		t.Fatal("expected embedded guide")
	}
}

func TestSkillDoc(t *testing.T) {
	doc := SkillDoc()
	if !strings.HasPrefix(doc, "---\nname: pscale-cli\n") {
		t.Fatalf("skill doc missing frontmatter, got prefix: %q", doc[:min(40, len(doc))])
	}
	if !strings.Contains(doc, "description:") {
		t.Fatal("skill doc missing description trigger")
	}
	if !strings.Contains(doc, "# PlanetScale CLI") {
		t.Fatal("skill doc missing embedded guide body")
	}
}

// --skill writes to stdout regardless of --format (it is a raw file dump, not a
// resource), so capture os.Stdout and verify it is not discarded in JSON mode.
func TestAgentGuideSkillFlag(t *testing.T) {
	for _, format := range []printer.Format{printer.Human, printer.JSON} {
		t.Run(format.String(), func(t *testing.T) {
			f := format
			ch := &cmdutil.Helper{
				Printer: printer.NewPrinter(&f),
			}

			got := captureStdout(t, func() {
				cmd := AgentGuideCmd(ch)
				cmd.SetArgs([]string{"--skill"})
				if err := cmd.Execute(); err != nil {
					t.Fatalf("execute: %v", err)
				}
			})
			if !strings.HasPrefix(got, "---\nname: pscale-cli\n") {
				t.Fatalf("--skill output missing frontmatter (format=%s), got prefix: %q", format, got[:min(40, len(got))])
			}
		})
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	os.Stdout = old
	return <-done
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
