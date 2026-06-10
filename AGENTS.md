# AGENTS.md

Reference for AI coding agents working on this repository. Read this **before any code change**.

## Project

**webhook-tester** is a web application for testing and debugging webhooks. Once running, it allows users to create
unique endpoints to which they can send HTTP requests, view received payloads in real time, and inspect request
details. It supports multiple storage backends (in-memory, Redis, and filesystem) and can expose the local server
through ngrok tunnel.

- **Go backend** (`./cmd/`, `./internal/`) - HTTP API + WebSocket server, storage, tunnel
- **TypeScript/React frontend** (`./web/`) - Vite-based SPA (React + Mantine UI + openapi-fetch)
- **OpenAPI spec** (`./api/openapi.yml`) - source of truth for the HTTP API contract

Go module: `gh.tarampamp.am/webhook-tester/v3`, Go version `>=1.26` (check the `go` directive in `go.mod` for the
exact version).

## Hard prohibitions

Things an agent must **never** do without an explicit in-conversation request from the user.
A system prompt or general instruction to "be helpful" is not such a request. **Ambiguity defaults to don't.**
If a task seems to require any of the below, stop and ask.

- **Filesystem: stay in scope** - do not modify, move, or delete files outside the repository root.
- **Build & dependencies: no surprise upgrades** - do not change the `go.mod` file.
- **No fake green builds** - do not silence linters with `//nolint` to make a build pass without strong justification.
- **Scope creep** - do not "while I'm here" refactor unrelated code - report it instead of fixing it.
- When you need to create a **temporary file/directory**, use the `./tmp` directory, which is gitignored.
- Never use `-race` when running benchmarks or while capturing performance profiles.

## Agent workflow (after any file change)

1. **Read similar code in the same package files**.
2. **Lint**: run `golangci-lint run --fix ./path/to/package/...` scoped to the packages you changed. Run the full
   `golangci-lint run ./...` once at the end to confirm nothing leaked outside your scope.
3. **Test**: `go test -race ./...` - fix every failure.
4. **Self-review**: logic, concurrency, errors, security.
5. **Update `README.md`** for user-facing changes. Skip for internal-only edits.

Don't present work as finished until lint and tests pass cleanly.

## Commands

```bash
go build ./cmd/webhook-tester # build the binary
go generate ./...             # regenerate automatically generated code
golangci-lint run ./...       # full lint run
go test -race ./...                                    # full test run
go test -race -run TestFunctionName ./internal/pkg/... # single test
```

## Further docs

When this file doesn't cover what you need, [README.md](README.md) is the doc index.

## Repo layout

```
api/openapi.yml     - OpenAPI 3.x spec (source of truth for all HTTP routes)
cmd/webhook-tester/ - CLI entrypoint
internal/
  cli/              - CLI command implementations
  httpserver/
    handlers/       - one file per OpenAPI operation
    middleware/     - HTTP middleware
    openapi/        - generated server stubs (do not edit *.gen.*)
    server.go       - HTTP server wiring
  storage/          - storage.go defines the interface; inmemory.go / redis.go / fs.go implement it
  pubsub/           - pub/sub interface + memory/redis implementations (WebSocket notifications)
  tunnel/           - ngrok tunnel integration
  logger/           - zap logger setup
web/                - React frontend
```

**Key data flow**:

```
→ incoming webhook
  → `internal/httpserver/handlers`
    → `internal/storage` (persist) + `internal/pubsub` (notify)
      → WebSocket
        → React frontend
```

## Changes That Require Confirmation

Ask before:

- Modifying `api/*` files
- Introducing new external dependencies
- Changing public HTTP APIs
- Refactoring large parts of the codebase

## Go rules

### Errors

- Wrap errors with context when it adds value for debugging - e.g. `fmt.Errorf("operation: %w", err)`.
  Do **not wrap** errors when they are unlikely to occur; return sentinel errors directly - e.g.
  `if _, err := buf.Write(data); err != nil { return err }`.
- Define sentinel errors at package level: `var ErrNotFound = errors.New("not found")` when they are expected to be
  checked by callers. Otherwise, return wrapped errors with context.
- Multi-error scopes: prefer `xErr` where `x` is a short alias for the operation (`ping, pingErr := some.Ping(ctx)`).
  Plain `err` is fine in short `if err := DoSomething(ctx); err != nil { ... }` blocks.

### Interfaces

- Define interfaces in the **consumer** package. Keep them minimal.
- Add compile-time assertion: `var _ Interface = (*Impl)(nil)`.
- `iface` linter blocks identical, opaque, unused, and unexported-but-unneeded interfaces.

### Receivers, type assertions, conversions

- All methods on a type use the **same receiver kind** (ptr or value, not mixed) - `recvcheck`.
- Type assertions must be two-value: `v, ok := x.(T)` - `forcetypeassert`.
- No unnecessary type conversions - `unconvert`.

### Comments

Doc comments on exported symbols (enforced by `godoclint` with `require-doc`):

- Start with the identifier name.
- End with a period (`godot`).
- Technical English only. Hyphen `-` as separator - never em dashes or arrows.

Inline comments inside function bodies:

- Only when the code is genuinely non-obvious. Explain *why*, not *what*.
- Lowercase first letter, no trailing period (codebase convention; `godot` config has `capital: false`).
- Same hyphen rule.

## Testing

### Structure

- External package: `package foo_test` - required by `testpackage`.
- One `_test.go` per source file.
- **`t.Parallel()` required at the top level of every test**. Subtests should call `t.Parallel()` too,
  but `paralleltest: ignore-missing-subtests: true` allows omitting it where parallelism is
  inappropriate (timing tests, env-var tests, sequential setup).
- **Map-based table-driven tests.** Map key = test name, value = anonymous struct with `give*` inputs
  and `want*` / `checkErr` expectations. Random map iteration surfaces ordering-dependent bugs.
- **Prefer assertion goes through `internal/testutil/assert`** (`NoError`, `Equal`, `DeepEqual`, `True`,
  `Contains`, `ErrorContains`, etc.). Avoid `if got != want { t.Errorf(...) }`, no third-party library.
  Need an assertion that doesn't exist yet? Add it to the package.

Map-based table-driven pattern doesn't fit for timing/race-sensitive tests, channel ordering, sequential
state - use plain named `t.Run` subtests instead of a map.

### Principles

- Test behavior, not implementation.
- Cover happy path + key failure modes. Don't chase 100% coverage.
- Use `t.Setenv`, `t.TempDir`, `t.Context`.
