# Component 6 — REST API

**Purpose**: External control surface - resource-oriented JSON endpoints for operators/orchestrators.

## Functional Requirements

| ID | Requirement |
|----|-------------|
| FR-1 | CRUD sessions |
| FR-2 | CRUD images |
| FR-3 | Network policy endpoints |
| FR-4 | Proxy status endpoint |
| FR-5 | Shared-folder status endpoint |
| FR-6 | Audit log query |
| FR-7 | Host health/capacity |
| FR-8 | Backend capabilities |
| FR-9 | Auth on mutating endpoints |
| FR-10 | Authorization enforcement |
| FR-11 | Record audit for mutating requests |
| FR-12 | Consistent JSON errors |
| FR-13 | API versioning (/v1/) |
| FR-14 | HTTPS |

## Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-1 | Latency < 10ms overhead |
| NFR-2 | 50+ concurrent connections |
| NFR-3 | 30s or 202 Accepted for long-running (POST /v1/sessions returns async) |
| NFR-4 | 1MB request body limit |
| NFR-5 | Graceful shutdown |
| NFR-6 | Consistent envelope `{"data":..., "error":...}` |

## Endpoints

| Method | Path | Auth | Notes |
|--------|------|------|-------|
| POST | /v1/sessions | operator | 202 Accepted |
| GET | /v1/sessions | required | list |
| GET | /v1/sessions/{id} | owner/operator | |
| POST | /v1/sessions/{id}/stop | owner/operator | |
| DELETE | /v1/sessions/{id} | owner/operator | |
| GET | /v1/sessions/{id}/connection | owner/operator | |
| GET | /v1/sessions/{id}/network | | |
| POST | /v1/sessions/{id}/network/* | operator | |
| GET | /v1/sessions/{id}/proxy | | |
| GET | /v1/sessions/{id}/shared-folder | | |
| GET | /v1/sessions/{id}/logs | owner/operator | |
| POST | /v1/images | operator | |
| GET | /v1/images | | |
| DELETE | /v1/images/{id} | operator | |
| GET | /v1/audit | operator | |
| GET | /v1/health | | |
| GET | /v1/capabilities | | |

## Implementation Notes

- **Router**: Go 1.22+ `http.ServeMux` (standard library)
- **Validation**: image_id (alphanum, max 64), owner (a-zA-Z0-9_-, max 64), cpus (1-32), memory_mb (512-65536), disk_gb (1-1024)
- **Rate limiting**: ADR deferred to production hardening
- **Long-running ops**: POST /v1/sessions returns 202, client polls GET /v1/sessions/{id}