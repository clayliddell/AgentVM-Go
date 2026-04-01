# Component 1 — VM Manager

**Purpose**: Owns VM lifecycle via libvirt (create, start, stop, destroy), manages images and overlays.

## Functional Requirements

| ID | Requirement |
|----|-------------|
| FR-1 | Import, list, inspect, delete base images |
| FR-2 | Report image capability metadata |
| FR-3 | Create per-session overlay (qcow2 copy-on-write) |
| FR-4 | Generate cloud-init seed ISO |
| FR-5 | Define libvirt domain XML |
| FR-6 | Start domain, wait for guest reachability |
| FR-7 | Gracefully stop or force-destroy domain |
| FR-8 | Undefine domain, remove artifacts |
| FR-9 | Query domain state, resources, connection info |
| FR-10 | Reconcile on restart, report orphans |
| FR-11 | Support nested virtualization |
| FR-12 | Emit lifecycle events to audit bus |

## Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-1 | Domain creation < 10s |
| NFR-2 | qcow2 copy-on-write overlay (no full copy) |
| NFR-3 | Go libvirt bindings or shell-out behind interface |
| NFR-4 | Domain XML deterministic and testable |
| NFR-5 | Fails closed if libvirt unreachable |
| NFR-6 | Cleanup idempotent |
| NFR-7 | Base images read-only |
| NFR-8 | No cross-feature state |

## Contracts

### LibvirtClient Interface
```go
type LibvirtClient interface {
    Connect() error
    DefineXML(xml string) error
    CreateDomain(xml string) error
    DestroyDomain(name string) error
    UndefineDomain(name string) error
    ListDomains() ([]Domain, error)
}
type Domain interface {
    Name() string
    Create() error
    GetState() (int, error)
    GetXMLDesc(flags int) (string, error)
}
```

### ShellOut Fallback
```go
type ShellExecutor interface {
    Exec(ctx context.Context, cmd string, args ...string) (string, error)
}
```

### Inbound (consumer defines)
```go
type VMManager interface {
    ImportImage(spec ImageImportSpec) (ImageID, error)
    ListImages() ([]ImageInfo, error)
    DeleteImage(id ImageID) error  // checks session store for active VMs
    CreateVM(spec VMSpec) (VMHandle, error)
    StartVM(handle VMHandle) (ConnectionInfo, error)
    StopVM(handle VMHandle, force bool) error
    DestroyVM(handle VMHandle) error
    Reconcile(knownSessions []SessionID) (OrphanReport, error)
}
```

## File Layout
```
internal/features/vm/image/
  types.go, service.go, store.go, errors.go
internal/features/vm/lifecycle/
  types.go, service.go, domain.go, libvirt.go, disk.go, cloudinit.go, errors.go
shared/types/
  image.go, vm.go  // canonical type definitions
```

## Implementation Notes

- **Domain XML**: Use typed struct + `encoding/xml`, not string concat
- **DeleteImage**: Query SessionStore interface to verify no active VMs reference image
- **ConnectionInfo**: `SSHAddress { Scheme, Host, Port, VSOCKCID }` - structured parsing
- **AuditEmitter**: Defined in shared/types/audit.go; VM Manager emits events