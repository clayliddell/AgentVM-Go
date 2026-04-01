# Phase 3: Public Control Plane

## Tasks

### REST API (Component 6)
- Implement sessions CRUD endpoints
- Implement images CRUD endpoints
- Implement network policy query endpoints
- Implement proxy status endpoint
- Implement shared-folder status endpoint
- Implement audit log query endpoint
- Implement host health/capacity endpoint
- Implement capabilities endpoint
- Implement authentication on mutating endpoints
- Implement authorization enforcement
- Implement request body validation rules
- Implement 202 Accepted for async session creation
- Implement /v1/sessions/{id}/logs endpoint
- Add per-endpoint auth matrix

### Observability - External (Component 7)
- Implement Prometheus /metrics endpoint
- Implement configurable log levels
- Implement timing metrics emission
- Fix metrics interface to use Prometheus conventions

## Functional Requirements

1. Sessions CRUD: POST (create), GET (read), GET (list), DELETE (destroy).
2. Images CRUD: POST (import), GET (list), GET (inspect), DELETE (remove).
3. Network policy query: GET current policy for a session.
4. Proxy status: GET proxy state and stats per session.
5. Shared-folder status: GET mount state and stats per session.
6. Audit log query: GET events filtered by type, session, time range.
7. Host health/capacity: GET host resource utilization and service status.
8. Capabilities: GET supported features, backends, and constraints.
9. All mutating endpoints (POST, PUT, DELETE, PATCH) require authentication.
10. Authorization enforced per endpoint; session-scoped operations require session ownership.
11. Request bodies validated: required fields, type constraints, range checks.
12. Session creation returns 202 Accepted with a polling location for async provisioning.
13. /v1/sessions/{id}/logs streams or returns VM console logs.
14. Per-endpoint auth matrix documented and enforced.
15. /metrics endpoint exposes Prometheus-format metrics (counters, gauges, histograms).
16. Log level configurable at runtime via config reload or signal.
17. Timing metrics emitted for: request latency, session create duration, VM boot time.

## Non-Functional Requirements

- All endpoints respond within 500ms (non-VM operations).
- 202 Accepted returned within 100ms for session creation.
- API follows REST conventions: proper HTTP methods, status codes, JSON bodies.
- Error responses include machine-readable error codes and human-readable messages.
- Prometheus metrics endpoint scrapable in < 50ms.
- API versioned under /v1/ prefix.
- Request validation errors return 400 with field-level detail.
- Authentication failures return 401; authorization failures return 403.

## E2E Test Conditions

1. POST /v1/sessions: verify 202 Accepted returned with polling URL.
2. GET /v1/sessions/{id}: verify session details returned after provisioning completes.
3. GET /v1/sessions: verify list returns all sessions; filter by status works.
4. DELETE /v1/sessions/{id}: verify session destroyed and 200/204 returned.
5. POST /v1/images: verify image imported; GET returns it in list.
6. GET /v1/images/{id}: verify inspect returns metadata (OS, arch, capabilities).
7. DELETE /v1/images/{id}: verify image removed.
8. GET /v1/sessions/{id}/network-policy: verify policy mode and rules returned.
9. GET /v1/sessions/{id}/proxy-status: verify proxy state and stats.
10. GET /v1/sessions/{id}/shared-folder-status: verify mount state.
11. GET /v1/audit: verify events returned; filter by session, type, time range.
12. GET /v1/health: verify host health and capacity info.
13. GET /v1/capabilities: verify feature list and constraints.
14. Call mutating endpoint without auth; verify 401 returned.
15. Call session-scoped endpoint as non-owner; verify 403 returned.
16. POST /v1/sessions with invalid body; verify 400 with field errors.
17. GET /v1/sessions/{id}/logs: verify console logs returned.
18. GET /metrics: verify Prometheus-format metrics returned.
19. Change log level at runtime; verify log output changes without restart.
20. Verify timing metrics appear for API requests and session operations.
21. Run go vet and project linters; zero violations.
