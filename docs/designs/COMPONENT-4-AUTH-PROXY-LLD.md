# Component 4 — Auth Proxy Manager

**Purpose**: Host-side secret mediation - replaces dummy credentials with real API keys for provider requests.

## Functional Requirements

| ID | Requirement |
|----|-------------|
| FR-1 | Start per-session proxy (virtio-vsock) |
| FR-2 | Accept guest requests at isolated endpoint |
| FR-3 | Validate session identity |
| FR-4 | Replace dummy auth header with real key |
| FR-5 | Forward request to upstream |
| FR-6 | Return response unmodified |
| FR-7 | Log request metadata (never real key) |
| FR-8 | Support per-session or global provider config |
| FR-9 | Stop proxy and cleanup on destroy |
| FR-10 | Reconcile on restart |
| FR-11 | Support multiple providers |

## Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-1 | Latency overhead < 50ms (p95) |
| NFR-2 | Process-level isolation |
| NFR-3 | Real keys in host secrets store only |
| NFR-4 | No admin API, no shell |
| NFR-5 | Fail closed on validation failure |
| NFR-6 | Handle ≥10 concurrent requests |
| NFR-7 | Memory < 50MB per instance |

## Contracts

```go
type ProxyLifecycle interface {
    StartProxy(sessionID SessionID, config ProxyConfig) (ProxyEndpoint, error)
    StopProxy(sessionID SessionID) error
    GetProxyStatus(sessionID SessionID) (ProxyStatus, error)
}
```

**ProviderConfig** (shared/types/proxy.go):
```go
type ProviderConfig struct {
    Name     string
    BaseURL  string
    APIKey   Sensitive  // wrapper preventing serialization
    DummyKey string
}
```

## Implementation Notes

- **Sensitive**: `type Sensitive string` with String()/MarshalJSON()/GoString() = "***REDACTED***"
- **Parent Process State**: ProxyStatus via shared file or localhost-only socket - no general HTTP surface
- **VSOCK Fallback**: If no vsock, use bridge IP + iptables forwarding
- **Concurrent Model**: Bounded worker pool (max 10)
- **Env Injection**: Proxy provides CloudInitEnvVars → Session Manager → VM Manager cloud-init