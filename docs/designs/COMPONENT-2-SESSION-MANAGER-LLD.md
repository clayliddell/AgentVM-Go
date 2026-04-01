# Component 2 — Session Manager

**Purpose**: Orchestration layer - coordinates VM, network, proxy, shared folder into session lifecycle.

## Functional Requirements

| ID | Requirement |
|----|-------------|
| FR-1 | Accept session request, return session ID |
| FR-2 | Orchestrate VM provisioning with rollback on failure |
| FR-3 | Enforce session state machine |
| FR-4 | Support force-stop/destroy |
| FR-5 | Persist to SQLite |
| FR-6 | Query/list sessions |
| FR-7 | Expose connection info |
| FR-8 | Reconcile on restart |
| FR-9 | Enforce capacity limits |
| FR-10 | Emit audit events |

## Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-1 | Session creation < 60s |
| NFR-2 | Atomic state transitions |
| NFR-3 | Rollback leaves no orphans |
| NFR-4 | Stateless beyond SQLite |
| NFR-5 | Serialize concurrent creates per session |
| NFR-6 | Fails closed, cleanup runs on error |

## Contracts

```go
type SessionService interface {
    CreateSession(req CreateSessionRequest) (SessionID, error)
    GetSession(id SessionID) (Session, error)
    ListSessions(filter SessionFilter) ([]Session, error)
    StopSession(id SessionID, force bool) error
    DestroySession(id SessionID, force bool) error
    GetConnectionInfo(id SessionID) (SessionConnectionInfo, error)
}
```

**Outbound Interfaces** (consumer defines):
```go
type VMProvisioner { CreateVM, StartVM, StopVM, DestroyVM, GetVMState }
type NetworkPolicyApplier { ApplyPolicy, RemovePolicy, GetPolicy }
type ProxyLifecycle { StartProxy, StopProxy, GetProxyStatus }
type SharedFolderManager { PrepareMount, CleanupMount }  // SharedFolderSpec in shared/types
type SessionStore { CreateSession, UpdateSession, GetSession, ListSessions }
```

**Types** (shared/types/session.go):
- `Session`, `SessionStatus` (requested→creating→running→stopping→destroyed/error)
- `SessionConnectionInfo { SSHInfo, ProxyEndpoint, SharedFolderMount }`
- `SSHInfo.PrivateKey`: ephemeral - returned once on creation, NOT persisted

## Implementation Notes

- **Provisioning Order**: PrepareMount→CreateVM→ApplyPolicy→StartProxy→StartVM. Justification: PrepareMount only creates host directory; mount happens at VM boot.
- **Concurrency**: Per-session mutex map in wiring/assembly.go
- **Rollback Failures**: Log error, mark session `error`, reconciliation cleans on restart
- **Reconciliation**: Host Resource Manager is primary coordinator; Session Manager implements session-specific convergence