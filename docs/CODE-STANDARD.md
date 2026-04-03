# Code Standard

## 1. Language & Toolchain

| Tool | Version | Purpose |
|------|---------|---------|
| Go | >=1.22 | Runtime |
| golangci-lint | >=1.57 | Linting |
| go test | built-in | Unit + integration + E2E tests |
| gosec | latest | Security scanning |
| govulncheck | latest | Vulnerability scanning |
| go-mutesting | latest | Mutation testing |

## 2. Linting & Formatting

```bash
golangci-lint run ./...
```

**Required linters:** `errcheck`, `govet`, `staticcheck`, `gosec`, `ineffassign`, `unused`, `gosimple`, `typecheck`, `gocritic`, `revive`.

**Custom analyzers** (see ARCHITECTURE.md):

| Rule | Analyzer | Purpose |
|------|----------|---------|
| R1 | `crossimport` | Reject cross-imports between feature packages |
| R2 | `sharedtypes` | Reject feature imports in shared/types |
| R3 | `wiringonly` | Only wiring may import multiple feature packages |
| R5 | `filesize` | Flag files exceeding 500 lines |
| R6 | `filecount` | Flag packages exceeding 10 non-test files |
| R8 | `circular` | Detect circular dependencies |
| R9 | `mutablestate` | Flag package-level mutable exported vars |
| R10 | `baninit` | Ban `init()` functions |
| R11/R12 | `importlocation` | Restrict `database/sql` to store.go, `net/http` to handler.go |
| R15 | `revive` (exported) | Require godoc on all exported declarations |
| R18 | `ioseparation` | Ban IO imports in business logic files |
| R19 | `reexport` | No re-exporting types from other module packages |

**Script-enforced rules:**

| Rule | Script | Purpose |
|------|--------|---------|
| R7 | `ci-budgets.sh` | Max 3 directory levels under internal/ |

**Not enforced (design principles):**

| Rule | Reason |
|------|--------|
| R4 | Consumer defines interfaces — design principle, enforced via code review |
| R13 | One primary type per package — not enforced |
| R14 | File naming conventions — not enforced |
| R16 | Feature README.md — not enforced |

## 3. Testing Requirements

### 3.1 Coverage Targets

| Test Type | Target | Enforcement |
|-----------|--------|-------------|
| Unit Test | >=95% line coverage | CI gate (`ci-test.sh`) |
| Mutation Test | >=90% mutation score | CI gate (`ci-mutation.sh`) |
| Integration Test | 100% of cross-feature contracts | CI gate (`ci-integration.sh`) |
| E2E Test | 100% of user-facing workflows | CI gate (`ci-e2e.sh`, gated by `CI_E2E=true`) |

### 3.2 Test Categories

**Unit Tests:**
- Test in isolation. Mock all external dependencies.
- Naming: `Test<Type>_<Method>_<Condition>_<ExpectedResult>`
- Run: `go test ./... -coverprofile=coverage.out`
- Features must not import other feature packages in tests.

**Mutation Tests:**
- Run on all unit-tested code.
- Target: 90% mutation score (killed / (killed + survived)).
- Run: `go-mutesting ./...`
- Surviving mutants must be documented as intentional or fixed.

**Integration Tests:**
- Test wiring-layer assembly with real or near-real dependencies.
- Every feature boundary must have at least one contract test.
- Run: `go test ./... -tags=integration`

**E2E Tests:**
- Test full user-facing workflows end to end against real infrastructure.
- Run: `go test ./... -tags=e2e`
- Require a real environment (staging or CI with provisioned infra).
- Every user workflow in the HLD must have at least one E2E test.

### 3.3 TDD Required

- No production code without a failing test first.
- Cycle: RED → GREEN → REFACTOR.
- PRs without corresponding tests are rejected.

## 4. API & Contract Compliance

- Every exported function must have a godoc comment (enforced by `revive`).
- Consumer-defined interfaces must document their purpose.
- Every feature `README.md` must list: purpose, entrypoints, dependencies.

## 5. Dependency & Security

```bash
govulncheck ./...
```

- Any `CRITICAL` or `HIGH` vulnerability blocks merge.
- No secrets in source code. Use environment variables or a secrets manager.
- All dependencies pinned via `go.mod` / `go.sum`.
- Dependency updates require a PR with changelog review.

## 6. Documentation

### 6.1 Per Feature

Each `features/<group>/<feature>/` directory must contain a `README.md`:
- Purpose (1 paragraph)
- Exported entrypoints
- Dependencies (which interfaces it defines, which it implements)

### 6.2 Architecture Decision Records

Deviations from `ARCHITECTURE.md` require an ADR in `docs/adr/`:

```
docs/adr/NNNN-short-title.md
```

- **Status:** Proposed | Accepted | Rejected | Superseded
- **Context:** Why the change is needed
- **Decision:** What was decided
- **Consequences:** Impact on architecture

## 7. CI/CD Pipeline

```
1. Lint          → golangci-lint + custom analyzers (R1, R2, R3, R5, R6, R8, R9, R10, R11, R12, R15, R18, R19)
2. Unit Tests    → go test -coverprofile, fail under 95%
3. Mutation      → go-mutesting, fail under 90%
4. Integration   → go test -tags=integration
5. Security      → gosec, govulncheck
6. Budgets       → file size (R5), file count (R6), directory depth (R7)
7. E2E           → go test -tags=e2e (staging only)
```

Fail-fast. Any stage failure stops the pipeline. No merge until all stages pass.

### 7.1 Pre-Commit Hooks

- `golangci-lint run --fix`
- File size / file count budget check

## 8. Performance & Reliability

- All operations must emit timing metrics.
- Exceeding latency budget triggers a warning log and metric alert.
- Before each release: concurrent load test, rapid create/destroy cycles, failure recovery verification.

## 9. Version Pinning

- Go dependencies pinned via `go.mod` and `go.sum`.
- System dependency versions documented in project README.
- Dependency updates require a PR with changelog review.
