package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/planetscale/cli/internal/cmdutil"
	"github.com/planetscale/cli/internal/migrate/d1"
)

func importD1ToolDefs() []ToolDef {
	return []ToolDef{
		{
			tool: mcp.NewTool("import_d1_doctor",
				mcp.WithDescription("Check prerequisites for Cloudflare D1 to PlanetScale Postgres import"),
			),
			handler: handleImportD1Doctor,
		},
		{
			tool: mcp.NewTool("import_d1_lint",
				mcp.WithDescription("Analyze a D1 SQL export for import issues"),
				mcp.WithString("input", mcp.Description("Path to D1 SQL export file"), mcp.Required()),
			),
			handler: handleImportD1Lint,
		},
		{
			tool: mcp.NewTool("import_d1_start",
				mcp.WithDescription("Start importing a D1 SQL export into PlanetScale Postgres (runs lint/plan first; dry_run previews without loading Postgres)"),
				mcp.WithString("input", mcp.Description("Path to D1 SQL export file"), mcp.Required()),
				mcp.WithString("org", mcp.Description("PlanetScale organization"), mcp.Required()),
				mcp.WithString("database", mcp.Description("PlanetScale database name"), mcp.Required()),
				mcp.WithString("branch", mcp.Description("PlanetScale branch name")),
				mcp.WithString("method", mcp.Description("Import method: pgloader (≥1GB) or psql (<1GB; schema via psql, data via pgloader)")),
				mcp.WithString("migration_id", mcp.Description("Migration ID from a prior start --dry-run")),
				mcp.WithBoolean("dry_run", mcp.Description("Lint and build import plan without loading Postgres")),
				mcp.WithBoolean("force", mcp.Description("Skip confirmations")),
			),
			handler: handleImportD1Start,
		},
		{
			tool: mcp.NewTool("import_d1_status",
				mcp.WithDescription("Show local D1 import state"),
				mcp.WithString("org", mcp.Description("PlanetScale organization"), mcp.Required()),
				mcp.WithString("database", mcp.Description("PlanetScale database name"), mcp.Required()),
				mcp.WithString("branch", mcp.Description("PlanetScale branch name")),
				mcp.WithString("migration_id", mcp.Description("Migration ID"), mcp.Required()),
			),
			handler: handleImportD1Status,
		},
		{
			tool: mcp.NewTool("import_d1_verify",
				mcp.WithDescription("Verify D1 import (row counts, sequences, coercion, content checks)"),
				mcp.WithString("org", mcp.Description("PlanetScale organization"), mcp.Required()),
				mcp.WithString("database", mcp.Description("PlanetScale database name"), mcp.Required()),
				mcp.WithString("branch", mcp.Description("PlanetScale branch name")),
				mcp.WithString("migration_id", mcp.Description("Migration ID")),
				mcp.WithString("input", mcp.Description("Path to D1 SQL export")),
				mcp.WithString("sqlite", mcp.Description("Path to local SQLite file")),
			),
			handler: handleImportD1Verify,
		},
	}
}

func handleImportD1Doctor(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error) {
	result, err := d1.Doctor(ctx)
	if err != nil {
		return importD1Error("doctor", err)
	}
	resp := d1.OKResponse("doctor", result, d1.DoctorNextSteps(result))
	if !result.Ready {
		return importD1Error("doctor", d1.DoctorReadinessError(result))
	}
	return importD1Result(resp)
}

func handleImportD1Lint(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error) {
	input, err := request.RequireString("input")
	if err != nil {
		return nil, err
	}
	result, err := d1.Lint(input)
	if err != nil {
		return importD1Error("lint", err)
	}
	resp := d1.LintResponse(result)
	return importD1Result(resp)
}

func handleImportD1Start(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error) {
	input, err := request.RequireString("input")
	if err != nil {
		return nil, err
	}
	org, err := request.RequireString("org")
	if err != nil {
		return nil, err
	}
	database, err := request.RequireString("database")
	if err != nil {
		return nil, err
	}
	branch := request.GetString("branch", "main")
	method := request.GetString("method", "")
	migrationID := request.GetString("migration_id", "")
	dryRun := request.GetBool("dry_run", false)

	importOpts := d1.ImportOptions{
		Org:         org,
		Database:    database,
		Branch:      branch,
		InputPath:   input,
		Method:      method,
		MigrationID: migrationID,
		DryRun:      dryRun,
	}

	prepared, err := d1.PrepareImport(importOpts)
	if err != nil {
		return importD1Error("start", err)
	}

	if !prepared.CanProceed {
		return importD1Result(d1.BlockedStartResponse(prepared, dryRun))
	}

	client, err := ch.Client()
	if err != nil {
		return nil, err
	}

	result, err := d1.Import(ctx, client, &d1.DefaultImportClient{Client: client}, importOpts, prepared)
	if err != nil {
		resp := d1.ErrorResponse("start", err)
		if result != nil {
			resp.Data = result
			if result.Lint != nil {
				resp.Issues = result.Lint.Issues
			}
		}
		resp.MigrationID = prepared.MigrationID
		return importD1Result(resp)
	}
	resp := d1.OKResponse("start", result, d1.StartNextSteps(result.MigrationID, database, result.Method, dryRun))
	resp.MigrationID = result.MigrationID
	if result.Lint != nil {
		resp.Issues = result.Lint.Issues
	}
	if dryRun {
		resp.Status = "dry_run"
	}
	return importD1Result(resp)
}

func handleImportD1Status(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error) {
	org, err := request.RequireString("org")
	if err != nil {
		return nil, err
	}
	database, err := request.RequireString("database")
	if err != nil {
		return nil, err
	}
	branch := request.GetString("branch", "main")
	migrationID, err := request.RequireString("migration_id")
	if err != nil {
		return nil, err
	}

	state, err := d1.Status(org, database, branch, migrationID)
	if err != nil {
		return importD1Error("status", err)
	}
	resp := d1.OKResponse("status", state, nil)
	resp.MigrationID = state.MigrationID
	return importD1Result(resp)
}

func handleImportD1Verify(ctx context.Context, request mcp.CallToolRequest, ch *cmdutil.Helper) (*mcp.CallToolResult, error) {
	org, err := request.RequireString("org")
	if err != nil {
		return nil, err
	}
	database, err := request.RequireString("database")
	if err != nil {
		return nil, err
	}
	branch := request.GetString("branch", "main")
	migrationID := request.GetString("migration_id", "")
	input := request.GetString("input", "")
	sqlitePath := request.GetString("sqlite", "")

	client, err := ch.Client()
	if err != nil {
		return nil, err
	}
	destURI, cleanup, err := d1.ResolveDestURI(ctx, client, d1.ImportOptions{
		Org:      org,
		Database: database,
		Branch:   branch,
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cleanup() }()

	result, err := d1.Verify(ctx, d1.VerifyOptions{
		Org:         org,
		Database:    database,
		Branch:      branch,
		MigrationID: migrationID,
		InputPath:   input,
		SQLitePath:  sqlitePath,
		DestURI:     destURI,
	})
	if err != nil {
		resp := d1.ErrorResponse("verify", err)
		if result != nil {
			resp.Data = result
		}
		return importD1Result(resp)
	}
	resp := d1.OKResponse("verify", result, nil)
	resp.MigrationID = migrationID
	return importD1Result(resp)
}

func importD1Result(resp d1.Response) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("marshal response: %w", err)
	}
	return mcp.NewToolResultText(string(b)), nil
}

func importD1Error(phase string, err error) (*mcp.CallToolResult, error) {
	resp := d1.ErrorResponse(phase, err)
	return importD1Result(resp)
}
