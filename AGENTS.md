# AGENTS - Project Rules

> Read this file AND the global rules before making any code changes -
> https://tarampampam.github.io/.github/ai/AGENTS.md (mirror -
> <https://raw.githubusercontent.com/tarampampam/.github/refs/heads/master/ai/AGENTS.md>).

## Instruction Priority

1. This file (`AGENTS.md` in this repository)
2. Global rules (external URLs)
3. Other documentation

If rules conflict, follow the highest priority source.

## Project Overview

**webhook-tester** is a web application for testing and debugging webhooks. It consists of:

- **Go backend** (`cmd/`, `internal/`) - HTTP API server, storage (in-memory/Redis/fs), WebSocket, ngrok tunnel support
- **TypeScript/React frontend** (`web/src/`) - Vite-based SPA (React + Mantine UI + openapi-fetch)
- **OpenAPI spec** (`api/openapi.yml`) - source of truth for the HTTP API contract

Go module: `gh.tarampamp.am/webhook-tester/v2`

## Architecture

```
cmd/webhook-tester/        - CLI entrypoint (urfave/cli/v3)
internal/
  cli/                     - CLI command implementations
  http/
    handlers/              - one file per OpenAPI operation
    middleware/            - HTTP middleware
    openapi/               - generated server stubs (do not edit *.gen.*)
    server.go              - HTTP server wiring
  storage/                 - storage.go defines the interface; inmemory.go / redis.go / fs.go implement it
  pubsub/                  - pub/sub interface + memory/redis implementations (WebSocket notifications)
  tunnel/                  - ngrok tunnel integration
  logger/                  - zap logger setup
web/
  src/
    api/                   - schema.gen.ts (generated) + openapi-fetch client
    screens/               - page-level React components (home, session, not-found)
    shared/                - reusable components, hooks, utils, providers
    routing/               - route definitions
    db/                    - client-side IndexedDB (Dexie) for local state
api/openapi.yml            - OpenAPI 3.x spec (source of truth for all HTTP routes)
```

**Key data flow**:

```
→ incoming webhook
  → `internal/http/handlers`
    → `internal/storage` (persist) + `internal/pubsub` (notify)
      → WebSocket
        → React frontend
```

## Key Commands

```bash
# Go - lint and test
golangci-lint run   # see .golangci.yml for active linters and settings (line-length 120, import order, etc.)
go test ./...

# Frontend - lint and test
npm --prefix ./web run lint   # runs tsc + eslint
npm --prefix ./web run test   # runs vitest

# Code generation (after changing api/openapi.yml or go:generate directives)
go generate -skip readme ./...
npm --prefix ./web run generate
```

> **Before making Go changes**: read `.golangci.yml` - it defines all active linters and their configuration.

Before submitting changes:

1. Regenerate code if needed
2. Run linters
3. Run tests

## Generated Files - Do Not Edit Manually

- `web/src/api/schema.gen.ts` - generated from `api/openapi.yml` via `npm --prefix ./web run generate`
- Any file matching `*.gen.*` or with a `Code generated` header

If the OpenAPI spec (`api/openapi.yml`) needs changes, edit it and then re-run codegen.

## Changes That Require Confirmation

Ask before:

- Modifying `api/*` files
- Changing storage interfaces or implementations
- Introducing new external dependencies
- Changing public HTTP APIs
- Refactoring large parts of the codebase

## Project-Specific Conventions

- Storage backends live in `internal/storage/` - in-memory, Redis and fs implementations share a common interface defined in `storage.go`.
- HTTP handlers are generated from the OpenAPI spec; do not add routes outside the spec without discussion.
- Logging uses `go.uber.org/zap`; follow existing patterns in `internal/logger/` and handler files.
- Frontend state management and component patterns follow what exists in `web/src/` - read existing components before writing new ones.
- Go import order (enforced by `gci` formatter): std → external → `gh.tarampamp.am/webhook-tester` internal.
- Use alias-based TS imports (e.g. `~/shared`) over relative (`../`) ones where the project is configured for it.

## Offline Fallback Rules

> Apply these only if the external rule URLs above are inaccessible. The external rules are authoritative.

### Go

- Wrap errors with context: `fmt.Errorf("operation: %w", err)`. Return sentinel errors directly when they are unlikely.
- Use `xErr` naming when multiple errors are in scope (e.g. `readErr`, `writeErr`); use `if err := ...; err != nil` for single short-lived errors.
- Interfaces in the consumer package; keep them minimal; add `var _ Interface = (*Impl)(nil)` compile-time assertions.
- Exported declarations must have a doc comment starting with the identifier name, ending with a period.
- No `fmt.Print*` / `print` / `println`; no global variables; no `init()` without justification.
- Line length ≤ 120 characters.
- Test files: `package foo_test` (external); one `_test.go` per tested file; both outer and inner `t.Parallel()`; map-based table-driven tests with `give*` / `want*` keys.

### TypeScript / React

- No `// eslint-disable*` or `// @ts-ignore` - fix the root cause instead.
- No braces omission in conditionals/loops.
- Prefer `unknown` over `any`; narrow types explicitly.
- React components: functional only, return type `React.JSX.Element`, PascalCase names.
- Hooks: `use` prefix, camelCase. Constants/enums: `SNAKE_UPPER_CASE`.
- CSS modules (`*.module.css`) for scoped styles.
- Tests: `describe` / `test` blocks; table-driven with `test.each` for repeated logic; cover happy path + key error cases.
