# Component 8 — Host Resource Manager

**Purpose**: cgroups v2 resource limits, capacity admission control, platform reconciliation.

## Functional Requirements

| ID | Requirement |
|----|-------------|
| FR-1 | Create cgroup v2 slice per session |
| FR-2 | Apply CPU limits |
| FR-3 | Apply memory limits |
| FR-4 | Apply PID limits |
| FR-5 | Apply I/O limits |
| FR-6 | Capacity admission control |
| FR-7 | Full platform reconciliation on startup |
| FR-8 | Expose host resources |
| FR-9 | Persist cgroup config |
| FR-10 | Configurable reservations |

## Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-1 | Cgroup creation < 500ms |
| NFR-2 | Capacity check < 100ms |
| NFR-3 | Reconcile < 30s |
| NFR-4 | No package-level mutable state |
| NFR-5 | Fail closed if cgroup fails |
| NFR-6 | Cleanup idempotent |
| NFR-7 | Prefer reattach over destroy |

## Contracts

```go
type ResourceEnforcer interface {
    CreateSlice(sessionID SessionID, limits ResourceLimits) error
    RemoveSlice(sessionID SessionID) error
    GetUsage(sessionID SessionID) (ResourceUsage, error)
    AssignProcess(sessionID SessionID, pid int) error
}

type CapacityManager interface {
    HasCapacity(req ResourceRequest) (bool, string, error)
    GetHostResources() (HostResources, error)
}

type Reconciler interface {
    Reconcile() (ReconcileReport, error)
}
```

## Implementation Notes

- **Reconciliation Ownership**: Host Resource Manager is primary coordinator; Session Manager implements session-specific convergence
- **ReconcileReport vs OrphanReport**: OrphanReport (VM-only from VM Manager) vs ReconcileReport (platform-wide from HRS). Host Resource Manager aggregates.
- **Cgroup Path**: `/sys/fs/cgroup/agentvm.slice/agentvm-session-<session-id>.slice/` - UUID format, no sanitization needed
- **AssignProcess Race**: Use libvirt `<resource>` element in domain XML for cgroup assignment, or immediate PID capture after StartVM
- **I/O Limits**: `io.max` requires device major:minor - query via libvirt block info or `/sys/dev/block/`

## File Layout
```
internal/features/resources/
  types.go, service.go, cgroup.go, reconcile.go, host.go, errors.go
```