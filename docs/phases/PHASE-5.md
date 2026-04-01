# Phase 5: Resource Enforcement & Reconciliation

## Tasks

### Host Resource Manager (Component 8)
- Implement cgroups v2 slice creation per session
- Implement CPU limits enforcement
- Implement memory limits enforcement
- Implement PID limits enforcement
- Implement I/O limits enforcement (device lookup strategy)
- Implement capacity admission control
- Implement full platform reconciliation on startup
- Implement host resources exposure
- Implement cgroup path sanitization
- Mitigate AssignProcess race condition
- Define single reconciliation coordinator

### Reconciliation Cross-Cutting
- Unify OrphanReport and ReconcileReport
- Implement VM console log capture (virDomainOpenConsole)
- Implement audit retention purge (90-day default)

## Functional Requirements

1. Each session gets a dedicated cgroups v2 slice (e.g., agentvm/session-{id}).
2. CPU limits enforced via cpu.max in the session's cgroup.
3. Memory limits enforced via memory.max in the session's cgroup.
4. PID limits enforced via pids.max in the session's cgroup.
5. I/O limits enforced via io.max; device major:minor resolved dynamically.
6. Capacity admission control rejects new sessions when host resources insufficient.
7. On startup, full platform reconciliation: VMs, cgroups, firewall, proxy, shared folders.
8. Host resources (CPU, memory, disk, capacity) queryable via API.
9. Cgroup paths are sanitized to prevent directory traversal or injection.
10. AssignProcess race condition mitigated (PID added to cgroup after fork, before exec).
11. Single reconciliation coordinator orchestrates all subsystem reconciliation on startup.
12. OrphanReport and ReconcileReport unified into a single report structure.
13. VM console logs captured via virDomainOpenConsole and stored per session.
14. Audit events older than 90 days purged automatically (configurable retention).

## Non-Functional Requirements

- Cgroup slice creation < 10ms per session.
- Resource limit changes take effect immediately (no restart required).
- Admission control decision < 50ms.
- Full platform reconciliation on startup < 30s for up to 50 sessions.
- Console log capture adds < 5% CPU overhead.
- Audit purge runs in background; does not block reads.
- I/O device lookup uses stable identifiers; survives device renumbering.

## E2E Test Conditions

1. Create a session; verify cgroup slice exists at expected path.
2. Set CPU limit; verify cpu.max reflects the limit; stress test confirms throttling.
3. Set memory limit; verify memory.max reflects the limit; OOM kills guest process (not host).
4. Set PID limit; verify pids.max reflects the limit; fork bomb is contained.
5. Set I/O limit; verify io.max reflects the limit; disk benchmark confirms throttling.
6. Attempt to create session exceeding host capacity; verify rejection with clear error.
7. Crash the service; restart; verify reconciliation detects and corrects all inconsistencies.
8. Query host resources; verify CPU, memory, disk, and capacity returned accurately.
9. Attempt path traversal in cgroup path; verify sanitization prevents it.
10. Verify PID is added to cgroup before process exec (no race window).
11. Verify unified reconciliation report covers VMs, cgroups, firewall, proxy, shared folders.
12. Capture VM console logs; verify logs stored and queryable per session.
13. Wait for or simulate audit events > 90 days old; verify purge removes them.
14. Verify purge does not block concurrent audit queries.
15. Run go vet and project linters; zero violations.
