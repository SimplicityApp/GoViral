# GoViral — Project Instructions

## Overview
GoViral is a monorepo containing a Go CLI tool for viral content creation on X and LinkedIn.
Phase 1 is the CLI (`apps/cli/`), Phase 2 will be a webapp (`apps/web/`).

## Conventions
- Go module path: `github.com/shuhao/goviral`
- Use `modernc.org/sqlite` (pure Go, no CGO)
- All shared data models in `pkg/models/`
- All business logic in `internal/`
- CLI-specific code only in `apps/cli/`
- Error handling: wrap errors with context using `fmt.Errorf("doing X: %w", err)`
- Use `lipgloss` for terminal output styling
- Config stored at `~/.goviral/config.yaml`

## Agent Team Roles

When using Agent Teams, the following ownership boundaries apply:

### Lead Agent
- Owns: `apps/cli/cmd/`, `apps/cli/main.go`, `internal/config/`, `internal/db/`, `pkg/models/`, `go.mod`, `go.sum`
- Responsibility: Project scaffolding, Cobra CLI wiring, config management, database layer, shared models, final integration

### Teammate: platform-apis
- Owns: `internal/platform/x/`, `internal/platform/linkedin/`
- Responsibility: X API v2 client, LinkedIn API client, all HTTP calls, auth handling, rate limiting
- Must use interfaces from `pkg/models/` for return types

### Teammate: ai-layer
- Owns: `internal/ai/claude/`, `internal/ai/persona/`, `internal/ai/generator/`
- Responsibility: Claude API client, persona analysis logic, content generation prompts and parsing
- Must use interfaces from `pkg/models/` for return types

### Teammate: testing
- Owns: all `*_test.go` files, `testdata/` directory
- Responsibility: Unit tests, integration tests, test fixtures, mocks
- Waits for other teammates to finish before writing tests

## Dependencies
- github.com/spf13/cobra (CLI framework)
- github.com/spf13/viper (config)
- modernc.org/sqlite (database)
- github.com/charmbracelet/lipgloss (terminal styling)
- gopkg.in/yaml.v3 (YAML parsing)
