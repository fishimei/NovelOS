# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development commands

- Download dependencies: `go mod download`
- Run the API server with the example config in PowerShell: ``$env:NOVEL_OS_CONFIG="deployments/config/config.example.yaml"; go run ./cmd/server``
- Build all packages: `go build ./...`
- Run all tests: `go test ./...`
- Run tests for one package: `go test ./internal/transport/http/handler`
- Run one test: `go test ./internal/transport/http/handler -run TestName`
- Run static checks used in this repo today: `go vet ./...`

## High-level architecture

NovelOS is a Go HTTP backend for an AI-assisted novel writing process. The product model is split into two phases:

1. **Setup flow**: turn a rough idea into structured project state.
   - Main resources: `projects`, `author-bible`, `characters`, `relationships`, `setup-sessions`, `setup-runs`
   - `setup-sessions/:id/advance` produces a `SetupRun`
   - `setup-runs/:id/result` returns a `SetupDraft`
   - `setup-sessions/:id/apply` is the point where selected draft output is accepted into persistent project state

2. **Story flow**: turn author prompts into draft chapters plus state updates.
   - Main resources: `story-sessions`, `story-runs`, `chapters`, `memories`
   - `story-sessions/:id/advance` produces a `StoryRun`
   - `story-runs/:id/events` is the SSE stream for generation progress
   - `story-runs/:id/commit` is the boundary where a draft becomes a committed chapter and its `MemoryPatch` is applied

The domain model for all of these resources lives in `internal/application/model/types.go`.

## Layering and code organization

- `cmd/server/main.go` is the process entrypoint. It reads `NOVEL_OS_CONFIG`, loads config, builds the app via `bootstrap.New`, and handles graceful shutdown.
- `internal/bootstrap/app.go` is the composition root. It wires config, repositories, concrete application use cases, HTTP handlers, router, and the `http.Server`.
- `internal/application/port/` contains the key interfaces:
  - repository interfaces for persistence
  - runtime contracts for SSE event streaming, transactions, clock, and ID generation
- `internal/application/service/` contains concrete application use cases only. Simple CRUD flows go directly from HTTP handlers to repository ports; cross-aggregate state changes such as setup-run apply and story-run commit live here.
- `internal/transport/http/` is the HTTP adapter layer:
  - `dto/` defines request binding structs
  - `handler/` maps HTTP requests into repository calls or concrete application use cases
  - `presenter/` defines the JSON response envelope
  - `middleware/` holds transport middleware such as request ID injection
- `internal/domain/status.go` contains the canonical session status, run status, and SSE event-name constants.

## HTTP/API conventions

- Routes are registered in `internal/transport/http/router.go` under `/api/v1`, plus `/healthz`.
- The API contract is described in `openapi.yaml`. When changing endpoint shapes or resource semantics, update the router/handlers and the OpenAPI spec together.
- Success responses use the presenter envelope `{ "data": ..., "meta": { "request_id": ... } }`.
- Paginated responses add `meta.pagination`.
- Error responses use `{ "error": { "code": ..., "message": ... }, "meta": { "request_id": ... } }`.
- `middleware.RequestID()` preserves `X-Request-Id` if present, otherwise generates one and echoes it back in the response header.
- Handler-level validation is intentionally thin: request parsing and simple normalization happen in `handler/common.go`; deeper business validation belongs in repositories or concrete application use cases depending on whether the rule is single-resource or cross-aggregate.

## Configuration

- Configuration shape is defined in `internal/config/config.go`.
- The app reads an optional config file path from `NOVEL_OS_CONFIG`.
- Environment overrides use the `NOVEL_OS_` prefix with `_` mapped to nested keys, for example `NOVEL_OS_POSTGRES_DSN`.
- Defaults are set in code; the example file is `deployments/config/config.example.yaml`.
- Current config sections are `app`, `postgres`, `ai`, and `sse`.

## Repo-specific note

- Root `.gitignore` ignores `*.md`, but `CLAUDE.md` is explicitly unignored so it can stay tracked.
