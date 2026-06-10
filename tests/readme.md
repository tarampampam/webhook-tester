# tests/

End-to-end feature tests. They compile and run the real application binary in-process, hit it over HTTP(S), and
assert on responses. No mocks, no stubs - the full stack runs.

## Layout

```
tests/
  feature/
    featuretest/      - shared helpers; regular Go package
    run_*_test.go     - top-level test functions
    group_*_test.go   - grouped scenarios that run against multiple backend configurations
```

## Running

```bash
go test ./tests/feature/...                # full suite
go test -run TestHTTPS ./tests/feature/... # TLS only
```

## Adding tests

- **New scenario inside the existing suite** - add a subtest to the appropriate `group_*.go` file. The scenario
  will automatically run against all three backends.
- **New group** - add a `group_*.go` file with a `groupTestXxx(t, baseUrl)` function, then call it from
  `runGroupTests` in `run_groups_test.go`.
- **New shared helper** - add it to `featuretest/`. Keep it in scope -`featuretest` is only for mechanics
  (HTTP, JSON, app lifecycle). Business-level assertions belong in the test files themselves.
