# Phase 1: Secure VM Lifecycle

## Tasks

### Wiring Layer (Component 0)
- Implement Config loading and validation
- Implement Assemble() function with interface wiring
- Implement SQLite migration runner and directory structure
- Implement concurrency serialization (per-session mutexes)
- Implement rollback failure handling

### VM Manager (Component 1)
- Define LibvirtClient interface and shell-out fallback
- Implement image import/list/inspect/delete
- Implement overlay disk creation (qcow2 copy-on-write)
- Implement cloud-init seed ISO generation
- Implement domain XML generation with typed struct
- Implement VM start/stop/destroy lifecycle
- Implement VM state query and connection info
- Implement VM reconciliation and orphan reporting
- Add domain.xml struct example to LLD

### Observability - Core (Component 7)
- Implement structured logging with correlation IDs
- Implement SQLite audit event store with WAL mode
- Implement audit event persistence and query
- Add logging vs audit failure semantics

## Functional Requirements

1. Config loads from file/env and validates all required fields before any component initializes.
2. Assemble() wires all component interfaces; missing dependencies fail fast at startup.
3. SQLite migrations run idempotently on startup; schema version tracked in DB.
4. Per-session mutexes serialize concurrent operations on the same session.
5. Rollback failures are logged and surfaced; system does not silently swallow errors.
6. LibvirtClient supports real libvirt and shell-out fallback behind a single interface.
7. Base images can be imported, listed, inspected, and deleted via VM Manager.
8. Overlay disks are created as qcow2 copy-on-write from a read-only base image.
9. Cloud-init seed ISOs generate user-data, meta-data, and network config.
10. Domain XML is generated from a typed Go struct (not string templates).
11. VM lifecycle: start, stop (graceful), destroy (force) all functional.
12. VM state and SSH connection info queryable at any time.
13. On startup, reconciliation detects orphaned libvirt domains and reports them.
14. Structured logs include correlation IDs linking related operations.
15. Audit events persist to SQLite in WAL mode; queryable by type, session, time range.
16. Audit write failures do not crash the process; log failures do not block audit writes.

## Non-Functional Requirements

- All feature packages follow boundary rules: no cross-imports, no package-level mutable state, no init().
- File size <= 500 lines (excluding tests); <= 10 .go files per package.
- Each package exports exactly one primary type.
- Config parsing completes in < 100ms.
- VM start-to-SSH-ready < 60s for standard images.
- Audit write latency < 5ms per event (WAL mode).
- Structured logging adds < 1ms overhead per log line.
- SQLite WAL checkpoint does not block readers.

## E2E Test Conditions

1. Start the service with a valid config; all components initialize without error.
2. Import a base image; verify it appears in list and inspect returns correct metadata.
3. Create an overlay disk from the imported image; verify qcow2 backing file chain is correct.
4. Generate a cloud-init seed ISO; verify it contains valid user-data and meta-data.
5. Start a VM from the overlay + cloud-init; verify domain exists in libvirt and reaches running state.
6. Query VM state and connection info; verify SSH host:port returned.
7. Stop the VM; verify domain transitions to shutoff.
8. Destroy the VM; verify domain removed from libvirt and overlay disk deleted.
9. Start service with orphaned domains present; verify reconciliation reports them.
10. Emit audit events during lifecycle; verify events queryable from SQLite.
11. Trigger a rollback failure; verify error is logged and surfaced (not swallowed).
12. Run concurrent operations on the same session; verify mutex serialization prevents races.
13. Run go vet and project linters; zero violations.
