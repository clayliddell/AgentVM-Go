# Component 5 — Shared Folder Manager

**Purpose**: Provides isolated host↔guest directory for file exchange.

## Functional Requirements

| ID | Requirement |
|----|-------------|
| FR-1 | Create unique host directory per session |
| FR-2 | Generate mount config (virtiofs preferred, 9p fallback) |
| FR-3 | Inject mount into cloud-init/VM XML |
| FR-4 | Prevent path traversal escape |
| FR-5 | Set host directory permissions 0700 |
| FR-6 | Support mount/unmount lifecycle |
| FR-7 | Cleanup on destroy |
| FR-8 | Report status |
| FR-9 | Emit audit events |
| FR-10 | Detect virtiofs support |

## Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-1 | Directory creation < 1s |
| NFR-2 | virtiofs near-native throughput |
| NFR-3 | No symlink escape |
| NFR-4 | Cleanup idempotent |
| NFR-5 | Host dir owned by libvirt/qemu, 0700 |
| NFR-6 | No state beyond filesystem |

## Contracts

```go
type SharedFolderManager interface {
    PrepareMount(sessionID SessionID, spec SharedFolderSpec) (SharedFolderMount, error)
    CleanupMount(sessionID SessionID) error
    GetMountStatus(sessionID SessionID) (SharedFolderStatus, error)
}
```

## Implementation Notes

- **Domain XML**: GenerateDomainXMLFragment in VM Manager (Component 1), not SharedFolder - features can't import features
- **Virtiofs Detection**: Check libvirt capabilities XML via `virConnectGetCapabilities()`, NOT `/dev/vhost-vsock`
- **MaxSizeMB**: Best-effort hint; virtiofs doesn't support native quotas; document as host quota enforcement or deferred
- **Types**: SharedFolderSpec, SharedFolderMount, MountType in shared/types/sharedfolder.go