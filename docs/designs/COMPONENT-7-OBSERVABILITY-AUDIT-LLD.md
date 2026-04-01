# Component 7 — Observability & Audit

**Purpose**: Structured logging, Prometheus metrics, health endpoints, append-only audit event store.

## Functional Requirements

| ID | Requirement |
|----|-------------|
| FR-1 | Emit structured JSON logs with correlation IDs |
| FR-2 | Prometheus /metrics endpoint |
| FR-3 | Record audit events |
| FR-4 | Persist events to SQLite |
| FR-5 | Host health check |
| FR-6 | Per-session health |
| FR-7 | VM console log capture |
| FR-8 | Host-only audit storage |
| FR-9 | Configurable log level |
| FR-10 | Emit timing metrics |

## Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-1 | Log emission < 1ms |
| NFR-2 | Audit write < 5ms (WAL mode) |
| NFR-3 | /metrics response < 100ms |
| NFR-4 | Audit storage ≤1GB without rotation |
| NFR-5 | No circular dependencies |
| NFR-6 | Logging fails silently (NOT audit) |

## Contracts

```go
type Logger interface { Debug, Info, Warn, Error, With(...Field) Logger }
type Metrics = *prometheus.Registry  // Use prometheus/client_golang directly
type AuditEmitter interface { EmitEvent(event AuditEvent) error }
type AuditStore interface { InsertEvent, QueryEvents }
type HealthService interface { GetHostHealth, GetSessionHealth }
```

## Implementation Notes

- **Metrics**: Use `prometheus/client_golang` directly, NOT custom wrapper (avoids label map mismatch)
- **Audit Failure**: Logging fails silently, but audit write MUST be logged and retried (not silent - violates HLD auditability)
- **Console Capture**: Use `virDomainOpenConsole`; buffer to `/var/lib/agentvm/sessions/<id>/console.log`
- **Retention Purge**: On startup + hourly; default 90 days
- **Logging vs Audit Split**: Logging is cross-cutting (shared package), Audit is feature (depends on logging, not vice versa)