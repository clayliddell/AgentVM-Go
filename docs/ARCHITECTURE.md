# Project Architecture: Feature-Isolated, Agent-Optimized Go Codebase

## Goal

Minimize the number of files an engineer (human or AI agent) must read
and touch to complete any given task. And minimize the number of tool
calls an AI agent must make. Every structural decision serves this goal.

---

## Directory Layout

```
internal/
  features/         # all business logic lives here
    <domain_group>/ # directory only, NOT a Go package
      <feature>/    # Go package — one feature per package
        types.go
        service.go
        store.go
        handler.go
        errors.go
        service_test.go
  shared/
    types/  # shared domain types only (UserID, Money)
    errors/ # common sentinel errors
  wiring/   # the ONLY package that connects features
    server.go
    config.go
```

Domain groups are for filesystem navigation only. Isolation rules apply
at the package level regardless of grouping.

---

## Boundary Rules

- Feature packages import only: `shared/types`, `stdlib`, and themselves.
- `shared/types` must not import any feature package.
- `wiring/` is the sole package permitted to import multiple features.
- Features define their own consumer interfaces. Never share interface
  definitions across features. Go's implicit interface satisfaction
  means the provider satisfies the interface without knowing the
  consumer exists.
- Interface definitions should declare only the minimum surface the
  consumer needs, not the provider's full API.

### How features communicate

The consuming feature defines a small interface and a projection type
representing the data it needs. The providing feature's concrete type
satisfies the interface implicitly. The wiring layer passes the
concrete instance during assembly. No cross-import occurs.

---

## Size Budgets

- 500 lines maximum per `.go` file (test files excluded).
- 10 `.go` files maximum per package (test files excluded).
- 3 directory levels maximum under `internal/`.

---

## Dependency Hygiene

- No circular dependencies.
- No package-level mutable state (`var x T = ...` at package scope).
- No `init()` functions.
- `database/sql` (and any DB driver) may only appear in `store.go` files.
- `net/http` may only appear in `handler.go` files.
- Business logic files must not perform IO. IO files must not contain
  business logic.

---

## Discoverability

- Each package exports exactly one primary type (`Service`, `Store`).
- File naming convention: `types.go`, `service.go`, `store.go`,
  `handler.go`, `errors.go`.
- Every exported function has a godoc comment.
- Each feature directory contains a `README.md` stating: purpose,
  exported entrypoints, and what it depends on.
- Error types are defined in the file containing the function that
  returns them.
- No re-exporting — package A must not export package B's types.

---

## Enforcement

The following rules require tooling. A skilled engineer should
implement them as linters or CI checks:

1. **Cross-import detection** — fail if any `features/` package
   imports another `features/` package. `dependency-cruiser`-style
   rule or a custom `go/analysis` analyzer.
2. **Circular dependency detection** — `go vet` or custom analyzer.
3. **Package-level state ban** — custom analyzer flagging top-level
   `var` declarations that are not `const` or function-scoped.
4. **`init()` ban** — custom analyzer or `golangci-lint` forbidigo rule.
5. **Import-location rules** — custom analyzer ensuring `database/sql`
   only appears in files named `store.go`, `net/http` only in
   `handler.go`.
6. **File size budget** — CI script or linter flagging files over 500
   lines.
7. **File count budget** — CI script flagging packages with more than
   10 non-test `.go` files.
8. **Missing godoc** — `golangci-lint` revive rule requiring comments
   on exported symbols.

---

## Testability

- Each feature can be tested in isolation by providing mock
  implementations of its consumer-defined interfaces.
- Mocks are 5–10 line structs in the test file. No mock generation
  frameworks needed.
- A feature's tests must not import another feature's package.
- The wiring layer has its own integration test that exercises the
  real assembly.

---

## What to Duplicate vs What to Share

- **Shared:** value types used across domains (`UserID`, `Money`,
  `Pagination`). One definition in `shared/types/`.
- **Duplicated:** small utility functions or validation logic that two
  features happen to need. Duplication is acceptable if it keeps the
  feature self-contained. Extract to `shared/` only when it's likely
  that three or more features will need it.
- **Never shared:** interfaces, business logic, error types between
  features.
