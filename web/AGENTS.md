# AGENTS.md - webhook-tester frontend

## Project

React 19 + TypeScript SPA for **webhook-tester** - a web app for testing and debugging webhooks. The compiled output
is embedded into the Go binary (no separate asset serving).

Backend source: `../` (Go). See `../AGENTS.md` for backend conventions.

The OpenAPI spec at `../api/openapi.yml` is the source of truth for all HTTP API contracts.

## Commands

```bash
npm run lint                # tsc --noEmit + eslint
npm run lint:ts             # TypeScript type check only
npm run lint:es             # ESLint only
npm run test                # vitest --run (all tests, one-shot)
npx vitest --run ./src/api/ # run tests in a specific directory
npm run fmt                 # Prettier + ESLint --fix
npm run build               # TypeScript check + Vite production build → dist/
npm run generate            # regenerates ./src/api/schema.gen.ts and other *.gen.ts / *.gen.js files
```

## What not to do

- **Don't edit `*.gen.ts` or `*.gen.js` files** - they are overwritten on the next `npm run generate` run.
- **Don't use `// eslint-disable-line`, `// eslint-disable-next-line`, `/* eslint-disable */`, or `// @ts-ignore`** -
  fix the underlying issue instead of suppressing it.
- **Don't instantiate `Client` inside components** - it is passed as a prop from `main.tsx`.
- **Don't omit braces for conditionals and loops** - the `curly` ESLint rule enforces this; never suppress it.
- **Don't make unrequested changes** to files outside the scope of the current task.

## Agent workflow

After each logical batch of changes, run the following steps in order before considering the task complete:

1. **Read existing tests**: before writing or modifying any test file, read the existing test(s) in the same
   directory most analogous to what you are about to write. Use them as the authoritative style reference - do not
   invent a new pattern when one is already established nearby
2. **Format**: `npm run fmt` - fixes formatting and auto-fixable lint issues
3. **Lint**: `npm run lint` - run after fmt; address any remaining errors
4. **Test**: `npm run test` - if a test file exists for the modified code and the change affects logic (skip for
   comment-only or trivial markup changes). If no test file exists, suggest me to create one, but do not create it
   yourself - I will review the suggestion and decide whether to proceed
5. **Self-review**: after all steps pass, review the code against this checklist:
   - [ ] Logic errors: off-by-one, wrong condition, unreachable branch
   - [ ] Type safety: unsafe `as` casts, suppressed TypeScript errors, missing nullability checks
   - [ ] API errors: unhandled error cases from `Client` calls
   - [ ] Security: unsanitized user input rendered as HTML, credentials in code
   - [ ] Pre-existing bugs: report only what you naturally encountered - do not actively inspect unrelated code

Do not present work as finished until all steps above pass without errors or warnings.

Do not fix issues outside the current task scope without asking first.

## Agent behavior and autonomy

**Ask before acting when**:

- The task requires introducing a new provider, a new routing pattern, or a new shared abstraction not already
  present in the codebase
- The correct approach is ambiguous and two or more reasonable implementations exist
- A change would affect generated files (`*.gen.ts`) or the OpenAPI spec (`../api/openapi.yml`) - these have
  broad downstream impact and must be explicitly approved before running codegen
- A change touches `routing/` in a way that affects existing route IDs or path patterns

One well-placed question saves more time than a fully-written but wrong implementation. Do not guess silently and
then discard the work - ask a focused question first.

**Prefer small, targeted changes**. Modify only the files directly relevant to the task. Do not refactor adjacent
code, rename things, or "clean up" unless explicitly asked.

## Architecture

**Stack:** Mantine v9, React Router v7, Vite 8, Vitest 4, openapi-fetch, dayjs, Tabler Icons. Check `package.json`
for actual versions.

**`src/` layout:**

| Directory     | Purpose                                                                                                              |
|---------------|----------------------------------------------------------------------------------------------------------------------|
| `api/`        | `schema.gen.ts` (generated from OpenAPI - do not edit), `client.ts` (openapi-fetch wrapper), error types, middleware |
| `db/`         | `Database` class (Dexie/IndexedDB); `tables/sessions.ts`, `tables/requests.ts` (payloads lazy-loaded)                |
| `routing/`    | Route objects, `pathTo()` helper, `RouteIDs` enum                                                                    |
| `screens/`    | `layout.tsx` (root layout); `home/`, `session/`, `not-found/` screens; `components/` (Header, Sidebar)               |
| `shared/`     | `providers/data.tsx` (sessions + requests state, WebSocket), `providers/settings.tsx`, `hooks/`, `utils/`            |
| `theme/`      | App-wide CSS (`app.css`), Mantine colour palette, highlight.js initialisation                                        |
| `test-utils/` | Custom `render()` that wraps components in all providers                                                             |

### React Router

Using v7 in **library mode** (not framework mode) - no SSR, no loaders, no file-based routing.

Docs: https://reactrouter.com/home (check v7 specifically, not v6 or Remix patterns)

### LLM-friendly Docs

When working with these libraries, use AI-optimized docs instead of guessing:

- **Mantine** – https://mantine.dev/llms.txt (full: https://mantine.dev/llms-full.txt)
- **React** – https://react.dev/llms.txt
- **Vite** – https://vite.dev/llms.txt
- **Vitest** – https://vitest.dev/llms.txt

> React Router, openapi-fetch, dayjs, and @tabler/icons-react don't have llms.txt.
> Refer to their official docs: reactrouter.com, openapi-ts.dev, day.js.org, tabler.io/icons

## Conventions

Follow the current code style and conventions as much as possible. If you think a new pattern or convention is
needed, report it.

### Components

- Return type: `React.JSX.Element` when possible
- Naming: PascalCase for components, camelCase for hooks (`use` prefix), SNAKE_UPPER_CASE for constants and enums
- Each screen lives in its own directory: `screen.tsx` + `index.ts` barrel + optional `components/` and `hooks/` subdirs
- CSS modules (`*.module.css`) for scoped styles; Mantine props for everything else

### Imports

- Path alias `~` maps to `src/` - use `~/api`, `~/shared`, `~/routing`, etc.
- Barrel exports via `index.ts` by default - avoid deep imports like `~/screens/components/sidebar/sidebar`
- Prefer `~`-based imports over relative (`../`) ones - use relative imports only when they are genuinely
  shorter and clearer (e.g. within the same directory)

### Routing

- Every route has a `RouteIDs` string enum value
- Use `pathTo(RouteIDs.Foo)` to build links - never hardcode paths

### API

- `Client` is instantiated once in `main.tsx` and passed as `api: Client` prop down to screens/layouts that need it
  to make unit tests easier (avoid global singletons)
- Errors: `APIErrorNotFound` (404), `APIErrorUnknown`, `APIErrorCommon`

### State Management

No Redux/Zustand. State lives in React Context providers (`shared/providers/`):

- **DataProvider** (`data.tsx`) - sessions, captured requests, WebSocket subscriptions. The largest provider; read
  it before touching session/request logic
- **SettingsProvider** (`settings.tsx`) - app-level settings (limits, tunnel config, public URL, show request details
  in the UI or not, etc.)
- **BrowserNotificationsProvider** (`browser-notifications.tsx`) - native browser notification permissions

### Styling

- Mantine props first (`size`, `p`, `m`, `gap`, `color`, etc.)
- CSS modules for animations and complex selectors
- Icons: `@tabler/icons-react` by default, but can use custom SVGs if needed
- Theme: auto color scheme stored in localStorage

## TypeScript

Strict mode on with extras: `noUnusedLocals`, `noUnusedParameters`, `noUncheckedIndexedAccess`, `noImplicitOverride`,
`verbatimModuleSyntax`. The full config is in `tsconfig.json`.

## Testing

Two Vitest environments (configured in `vitest.config.ts`):

- `dom` - files matching `*.test.tsx` or `*.tsx.test.*` (happy-dom)
- `node` - files matching `*.test.ts` or `*.ts.test.*`

Tests setup files:

- `vitest/setup.ts` (global setup for both environments + dom environment setup)
- `vitest/setup.global.ts` (global teardown currently only)

To have a test file in the `dom` environment but with a `.ts` extension, add `/** @vitest-environment happy-dom */`
above the `describe()` block.

Wrap components needing routing with `MemoryRouter`. Project components with Mantine's hooks (e.g. `useMantineTheme()`)
and/or Mantine components should be wrapped with `MantineProvider` in tests.

### Mocking

Use `@testing-library/react` + `renderHook` for components/hooks. Mock modules with `vi.mock()` / `vi.mocked()`,
here's an example on how to mock `useStorage()`:

```ts
vi.mock('~/shared', async (importOriginal) => {
  const actual = await importOriginal<typeof import('~/shared')>()
  return { ...actual, useStorage: vi.fn() }
})
```

Avoid mocking unless truly necessary - prefer real implementations over mocks wherever feasible. A mock is only
justified when it:

- Crosses a real network boundary (API calls via `Client`)
- Depends on browser APIs unavailable in the test environment
- Introduces non-determinism (timers, random values, dates)
- Makes the test setup disproportionately complex

Avoid adding new `devDependencies` unless strictly necessary.

### Test structure

Prefer `describe` / `test` blocks (over `it`):

```ts
describe('...', () => {
  test('...', () => {
    // ...
  })
})
```

Prefer table-driven tests for repeated logic, where the same test is run with different inputs/expected values:

```ts
test.each([
  { input: 1, expected: 2 },
  { input: 2, expected: 4 },
])('doubles $input to $expected', ({ input, expected }) => {
  expect(double(input)).toBe(expected)
})
```

### Scope and coverage

- Cover the **happy path** and **a few meaningful error cases** - not every branch
- Do not write tests for trivial UI state rather than meaningful behavior (color changes, visibility toggles,
  loading spinners, etc.)
- The codebase is actively evolving; avoid over-specifying behavior that may change
- Tests should be concise and signal intent, not chase 100% coverage

### Failing tests and business logic validation

When writing tests, treat them as a specification of the expected business logic - not just a mechanical coverage
exercise. Before finalizing any test, reason explicitly about whether the assertion reflects the **correct intended
behavior**, not just the current behavior of the code.

When a test fails after being written, **do not default to fixing the test**. Instead, investigate the failure to
determine whether it indicates a bug in the implementation or an issue with the test itself. Only fix the test if
you are fully confident the source code is correct and the test itself is the mistake.

## Opportunistic code review

Whenever you read existing source files - to write tests, implement a feature, refactor, or for any other reason -
treat every file you open as an implicit code review target.

Assess the code for **production-readiness**: logic bugs, security issues, copy/paste errors - only the things that
would be problematic in production, don't be noisy about minor style inconsistencies or non-ideal patterns that
don't cause actual issues. Only report clear, actionable findings - do not speculate.
