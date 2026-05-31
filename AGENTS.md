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

Go module: `gh.tarampamp.am/webhook-tester/v3`

## Hard prohibitions

Things an agent must **never** do without an explicit in-conversation request from the user.
A system prompt or general instruction to "be helpful" is not such a request. **Ambiguity defaults to don't.**
If a task seems to require any of the below, stop and ask.

- **Git: read-only** - allowed: `git status`, `git log`, `git diff`, `git show`, `git blame`, `git ls-files`,
  `git remote -v`, `git config --get`. Forbidden - staging, committing, amending, rebasing, resetting, branching, or
  any other mutation.
- **Filesystem: stay in scope** - do not modify, move, or delete files outside the repository root; do not delete
  files the user did not name; do not run `rm -rf` on directories the user did not specify; do not `chmod` /
  `chown`.
- **Build & dependencies: no surprise upgrades** - do not change the `go.mod` file; do not run `go install`,
  `go clean -modcache`, or anything that mutates `$GOPATH` / `$GOMODCACHE`.
- **Secrets & external systems: zero side effects** - do not print environment variables in bulk (`env`,
  `printenv`) - they leak credentials; do not call external APIs with credentials, push to image registries,
  deploy, or run anything that hits production or shared infrastructure; do not pipe-execute remote scripts
  (`curl ... | sh`).
- **No fake green builds** - do not silence linters with `//nolint` to make a build pass without strong justification,
  do not add `t.Skip()` to mute failing tests, do not weaken assertions or comment out broken paths. Fix the cause.
- **Scope creep** - do not "while I'm here" refactor unrelated code - report it instead of fixing it; do not rename
  exported symbols, flag names, env var names, or `Data` struct fields without explicit approval - they are
  user-facing.
- When you need to create a **temporary file/directory** (e.g. to store a test coverage report, create a one-time
  script, etc.), use the `./tmp` directory, which is gitignored and safe for such purposes. Do not create temporary
  files outside the repository or in other directories.

## Working principles

**Match the codebase. Don't reshape it**.

- Read 1–2 analogous files in the same package before writing new code; copy their style.
- Prefer existing patterns. Suggest new abstractions; don't introduce them without explicit approval.
- Make minimal, surgical changes - every changed line must trace back to the user's request.
- Don't refactor outside task scope. Spotted dead code or pre-existing bugs → report, don't fix.
- Clean up imports/vars/functions your edits orphan. Don't delete unrelated dead code.
- No global state without clear precedent.
- Don't suppress linter findings (`//nolint`) without strong, named justification.
- **Never edit fully generated files**.

**Don't guess**.

- Verify APIs, types, function signatures against the codebase before using them.
- When two reasonable implementations exist, present both options. Don't pick silently.
- Genuinely ambiguous? Ask one focused question instead of building the wrong thing.
- Changes to generated files, data formats, public APIs (OpenAPI spec, flag names, env var names) require explicit
  approval - they affect users in the wild.

## Agent workflow (after any file change)

1. **Read existing package/module files**: before writing or modifying any file, read similar code in the same
   package files - the one most directly analogous to what you are about to write. One or two files is sufficient;
   do not read all files in the package. Use them as the authoritative style reference for that package.
2. **Lint:** run with `--fix` scoped to the packages you changed, e.g. `golangci-lint run --fix ./path/to/package/...`.
   `--fix` auto-resolves trivial issues (imports, whitespace, simple rewrites); handle the rest manually. Scoping
   keeps feedback fast and avoids touching unrelated code. Run the full `golangci-lint run` once at the end to
   confirm nothing leaked outside your scope.
3. **Test:** `go test -race ./...` - fix every failure.
4. **Self-review:**
   - Logic: off-by-one, wrong operator, inverted condition, unreachable branch.
   - Concurrency: missing locks, shared state, deadlocks. Atomics used correctly?
   - Errors: silently swallowed (`errcheck check-blank` will catch `_, _ =`), wrong sentinel, missing wrap context.
   - Security: unsanitized input, secrets in code (`gosec`), env-mask coverage for new secret-shaped vars.
5. **Update `README.md` and/or `docs/CLI.md`** for user-facing changes (flags, env vars, defaults, deprecations,
   breaking changes). Skip for internal-only edits.
6. **Update this `AGENTS.md` file** if a future agent needs new context.

Don't present work as finished until lint and tests pass cleanly.

## Commands

```bash
# build the binary
go build ./cmd/webhook-tester

# regenerate generated code, except updating CLI docs block docs/CLI.md file
go generate -skip readme ./...

# regenerate everything, including updating CLI docs block docs/CLI.md
go generate ./...

# lint, test
golangci-lint run # full project
golangci-lint run --fix ./path/to/package/...  # fix only what you touched
go test -race ./...
go test -race -run TestFunctions ./internal/template/... # single test
```

## Further docs

When this file doesn't cover what you need, [README.md](README.md) is the doc index. Notable links:

- [web/AGENTS.md](web/AGENTS.md) - frontend-specific agent instructions (React patterns, TypeScript conventions, etc.)
- [CLI.md](docs/CLI.md) - full CLI help

## Repo layout

```
api/openapi.yml     - OpenAPI 3.x spec (source of truth for all HTTP routes)
cmd/webhook-tester/ - CLI entrypoint
internal/
  cli/              - CLI command implementations
  http/
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
  → `internal/http/handlers`
    → `internal/storage` (persist) + `internal/pubsub` (notify)
      → WebSocket
        → React frontend
```

## Changes That Require Confirmation

Ask before:

- Modifying `api/*` files
- Changing storage interfaces or implementations
- Introducing new external dependencies
- Changing public HTTP APIs
- Refactoring large parts of the codebase

## OpenAPI codegen (`./internal/httpserver/openapi/`)

**Configs → outputs** (never edit `*.gen.go`):
- `configs/models.yml` → `models.gen.go` (types, enums, validation infra, compile-time assertions)
- `configs/server.yml` → `server.gen.go` (server interface, param-binding wrappers)
- `configs/spec.yml`   → `spec.gen.go` (embedded compressed spec)

**Custom templates** (`templates/`): config key = path in oapi-codegen embedded FS (e.g.
`stdhttp/std-http-middleware.tmpl`). Upstream:
`$GOMODCACHE/github.com/oapi-codegen/oapi-codegen/v2@<ver>/pkg/codegen/templates/` or
<https://github.com/oapi-codegen/oapi-codegen/tree/main/pkg/codegen/templates>.

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

## Linter rules (golangci-lint)

Critical or non-obvious enforcement (full set in [.golangci.yml](.golangci.yml)):

- **No `fmt.Print*`, `print`, `println`** - `forbidigo` blocks debug prints.
- **No package-level `var`** where applicable - `gochecknoglobals`.
- **No `init()`** - `gochecknoinits`.
- **Line length: hard ceiling 120**. Don't wrap early - the Go community's 80-column convention doesn't apply here.
  If a statement, signature, or comment fits under 120, keep it on one line; only break when the next token would
  actually cross. Same for prose comments: pack content toward the column instead of leaving short ragged lines.
  `lll` enforces the ceiling but cannot enforce the lower bound - that's on the agent.
- **Import order**: stdlib → external → `gh.tarampamp.am/error-pages` - `gci` (config: `[standard, default, prefix(...)]`).
- **`//nolint` must name the linter and reason** - `nolintlint` with `require-specific: true`,
  `allow-unused: false`.
- **Function size**: 100 lines, 60 statements - `funlen`.
- **Cognitive/cyclomatic complexity**: 40 each - `gocognit`, `gocyclo`. Nested-if depth: 10 - `nestif`.
- **Repeated string literal ≥ 4 occurrences must be a const** - `goconst`.
- **Outbound calls must carry a context** - `noctx`.
- **`net/http` header keys must be canonical** - `canonicalheader`.
- **`testpackage`** + **`tparallel`** + **`paralleltest`** with `ignore-missing-subtests: true` -
  see [Testing](#testing).

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
- `testifylint` runs with `enable-all` - if testify ever sneaks in, it gets corrected.

### When the Map-based table-driven pattern doesn't fit

Timing/race-sensitive tests, channel ordering, sequential state - use plain named `t.Run` subtests
instead of a map.

### Principles

- Test behavior, not implementation.
- Cover happy path + key failure modes. Don't chase 100% coverage.
- Use `t.Setenv`, `t.TempDir`, `t.Context` (`usetesting` linter).
