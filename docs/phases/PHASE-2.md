# Phase 2: Session Model

## Tasks

### Shared Folder Manager (Component 5)
- Implement per-session host directory creation (0700)
- Implement virtiofs detection (via libvirt capabilities)
- Implement 9p fallback mechanism
- Implement mount config injection for cloud-init
- Implement path traversal prevention
- Implement mount cleanup on destroy

### Auth Proxy Manager (Component 4)
- Implement per-session proxy start (virtio-vsock)
- Implement session identity validation
- Implement dummy-to-real credential replacement
- Implement request forwarding and response passthrough
- Add APIKey Sensitive type wrapper
- Implement VSOCK fallback (bridge IP + iptables)
- Define env var injection contract for cloud-init

### Session Manager (Component 2)
- Implement CreateSession with orchestration
- Implement session state machine (requested→creating→running→stopping→destroyed/error)
- Implement rollback on provisioning failure
- Implement stop/destroy with force option
- Implement SQLite persistence
- Implement session list/query
- Implement connection info exposure
- Rename ConnectionInfo to SessionConnectionInfo
- Add concurrency serialization strategy

### Observability - Health (Component 7)
- Implement host health check endpoint
- Implement per-session health check
- Implement health status aggregation

## Functional Requirements

1. Each session gets a unique host directory with 0700 permissions owned by the service user.
2. virtiofs is detected via libvirt capabilities; used when available.
3. 9p is used as fallback when virtiofs is unsupported; selection is transparent to callers.
4. Cloud-init receives mount configuration (fstab entries or systemd units) for the shared folder.
5. Path traversal attempts (../, symlinks escaping the directory) are detected and blocked.
6. Shared folder state is cleaned up on session destroy (directory removed, mount config invalidated).
7. Auth proxy starts per-session, bound to virtio-vsock or fallback bridge IP.
8. Session identity is validated on every proxy request (session ID + VM channel identity).
9. Dummy credentials are replaced with real provider credentials before forwarding upstream.
10. Proxy forwards requests and passes responses transparently; no response body modification.
11. APIKey type uses redacted String() to prevent accidental log exposure.
12. VSOCK fallback uses bridge IP + iptables when virtio-vsock is unavailable.
13. Proxy env vars (base URL, dummy credentials) are injected via cloud-init.
14. CreateSession orchestrates: VM creation + shared folder + proxy + network config atomically.
15. Session state machine enforces valid transitions; invalid transitions return errors.
16. Provisioning failure triggers rollback: VM destroyed, shared folder cleaned, proxy stopped.
17. Stop is graceful (ACPI shutdown); destroy is force (hard kill). Both are supported.
18. Session state persists to SQLite; survives service restarts.
19. Sessions can be listed and filtered by status, owner.
20. Connection info (SSH, proxy endpoint, shared folder mount) exposed per session.
21. Host health check reports service status, libvirt connectivity, disk/capacity.
22. Per-session health check reports VM state, proxy state, shared folder mount state.
23. Health status aggregates across all sessions for a single query.

## Non-Functional Requirements

- Shared folder directory creation < 5ms.
- Auth proxy startup < 2s per session.
- Session creation orchestration < 90s end-to-end (image-dependent).
- Rollback completes in < 30s regardless of failure point.
- Session state transitions are atomic in SQLite (single transaction).
- Health check response < 100ms for host-level; < 500ms for per-session.
- Proxy request forwarding adds < 10ms latency overhead.

## E2E Test Conditions

1. Create a session; verify host directory exists with 0700 permissions.
2. With virtiofs available: verify guest mounts shared folder via virtiofs.
3. Without virtiofs: verify fallback to 9p and guest mount succeeds.
4. Attempt path traversal from guest (../etc/passwd); verify access is denied.
5. Destroy session; verify shared folder directory removed from host.
6. Start auth proxy for a session; verify it listens on expected vsock port.
7. Send a request with dummy credentials through proxy; verify real credentials used upstream.
8. Verify proxy request/response passthrough is transparent (no body modification).
9. Log an APIKey value; verify output is redacted.
10. Simulate VSOCK unavailability; verify fallback to bridge IP + iptables.
11. Create a session end-to-end: VM + shared folder + proxy all provisioned atomically.
12. Trigger provisioning failure mid-create; verify rollback cleans up all partial state.
13. Transition session through full state machine: requested→creating→running→stopping→destroyed.
14. Attempt invalid state transition (e.g., running→creating); verify error returned.
15. Restart service; verify session state loaded correctly from SQLite.
16. List sessions with filters; verify correct results.
17. Query connection info; verify SSH, proxy, and shared folder details returned.
18. Query host health; verify libvirt, disk, and capacity status.
19. Query per-session health; verify VM, proxy, and shared folder states.
20. Query aggregated health; verify correct rollup across all sessions.
21. Run go vet and project linters; zero violations.
