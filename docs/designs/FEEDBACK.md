# LLD Review Feedback

Reviewer note: All LLDs reviewed against HLD functional/non-functional requirements and ARCHITECTURE.md conventions. Reviews prioritize AI agent implementation success probability.

---

## Cross-Cutting Issues

- **No wiring layer spec.** ARCHITECTURE.md mandates `wiring/` as "the sole package that connects features," yet no LLD defines the wiring layer's structure, assembly logic, or how feature interfaces are wired to concrete implementations. AI agents will not infer this. A dedicated wiring specification or a concrete example in one LLD is required.
- **gRPC contracts absent.** HLD Section 16.2 and Section 19 require internal gRPC contracts for future extensibility. No LLD defines `.proto` files, service boundaries, or a gRPC server skeleton. Either include gRPC in the LLDs or explicitly defer with an ADR.
- **Duplicate type definitions.** Several LLDs define types inline (e.g., `VMManager` interface in LLD-1, `SessionService` in LLD-2 and LLD-6) that also appear in `shared/types/`. AI agents will be confused about which is canonical. Each LLD should clearly mark: "defined in `shared/types/`, shown here for reference" or remove inline redefinitions.
- **AuditEmitter duplication.** LLD-1, LLD-4, and LLD-7 each define their own `AuditEmitter` interface. If these are meant to be the same contract, define it once in `shared/` and reference it. If they differ, explain why.
- **SQLite migration strategy missing.** Multiple components (1, 2, 7, 8) use SQLite but no LLD defines schema migration infrastructure. AI agents need at minimum: a migration directory structure, version numbering, and a startup migration runner.
- **Rollback failure semantics unclear.** LLD-2 describes rollback on creation failure but does not specify what happens if rollback itself partially fails (e.g., VM destroy fails during cleanup). HLD requires "deterministic cleanup and reconciliation after crashes." Add explicit rollback-failure handling: reconcile on next startup.
- **Missing `/v1/sessions/{id}/logs` endpoint.** LLD-7 (User Story 4.3) references a `GET /v1/sessions/{id}/logs` endpoint, but LLD-6's endpoint table does not include it. Add it to the REST API spec.
- **NFR-3 in REST API conflicts with Session Manager.** LLD-6 NFR-3 says "all endpoints return within 30 s" but LLD-2 NFR-1 allows 60 s for session creation. LLD-6 should explicitly document that `POST /v1/sessions` returns `202 Accepted` with async polling, not block for 60 s.

---

## Component 1 — VM Manager

- **`LibvirtClient` interface not formally defined.** LLD-1 references `libvirt.go` as a "libvirt client wrapper (implements LibvirtClient interface)" but never defines `LibvirtClient`. AI agents cannot implement against an absent contract. Define it in `types.go` or `libvirt.go` with all methods used by `service.go`.
- **Shell-out wrapper undefined.** NFR-3 allows "a well-scoped shell-out wrapper behind an interface" as fallback for Go libvirt bindings. No interface or strategy is specified. If this fallback is needed, define the wrapper interface with method signatures.
- **`domain.go` uses `encoding/xml` struct approach without examples.** The LLD says "use Go `encoding/xml` with a typed struct (no string concatenation)" but provides no sample struct definition. AI agents benefit from at least one example fragment showing the struct-to-XML pattern.
- **Image store deletes lack safety clarification.** The `DeleteImage` function should "verify no active VMs reference it" but the LLD does not specify how this cross-reference check works without importing the session package. Define a query interface or pass a dependency.
- **`ConnectionInfo` uses string for SSHAddress.** Using a plain string for `SSHAddress` ("host-ip:port" or "vsock:<cid>:22") is fragile for AI parsing. Consider a struct with `Scheme`, `Host`, `Port` fields or at minimum document the exact format grammar.

---

## Component 2 — Session Manager

- **`ConnectionInfo` type conflict.** LLD-2 defines `ConnectionInfo` with `SSHInfo`, `ProxyEndpoint`, `SharedFolderMount` fields. LLD-1 also defines `ConnectionInfo` with `SSHAddress`, `VSOCKCID`, `ConsolePath`. These are different types with the same name. AI agents will fail to resolve which is which. Rename one (e.g., `SessionConnectionInfo` vs `VMConnectionInfo`).
- **`SharedFolderManager` interface is defined by Session Manager but `SharedFolderSpec` type is undefined.** The `PrepareMount` method takes `SharedFolderSpec` but this type is not shown in the Session Manager's data types. Cross-reference LLD-5 explicitly.
- **Provisioning order needs justification.** Step 1 calls `SharedFolderManager.PrepareMount` before `VMProvisioner.CreateVM`, but the shared folder mount requires the VM domain XML to be built. Clarify: does `PrepareMount` only create the host directory and return config, with actual mount happening during VM start? This matters for AI agents.
- **Concurrency serialization undefined.** NFR-5 says "concurrent session creation requests must be serialized per-session." No locking mechanism is specified (mutex map, channel-based serializer, SQLite lock). AI agents need a concrete strategy.
- **`SSHInfo.PrivateKey` in session struct.** The LLD says the private key is "returned only once, on creation" but the `Session` struct in `shared/types` includes `PrivateKey` as a persistent field. AI agents will store it in SQLite. Clarify: is it ephemeral-only, or persisted encrypted?

---

## Component 3 — Network Policy Engine

- **DNS resolution approach underspecified.** NFR-4 requires DNS re-resolution on TTL expiry, but the implementation plan uses a fixed 5-minute interval. These are contradictory. Either track DNS TTLs or document that the 5-minute interval is a deliberate simplification with an ADR.
- **`FirewallBackend` interface placement.** The interface is defined inside the network package, but for testability AI agents need to know whether it lives in `types.go`, `backend.go`, or its own file. The file layout lists `backend.go` for both interface and nftables implementation — separate the interface definition from the implementation to avoid AI agents coupling them.
- **Interaction between strict mode and private-range blocking unclear.** In strict mode, is the default DROP applied before or after the mandatory private-range block rules? If strict mode already drops everything, the private-range blocks are redundant. Document the rule ordering explicitly.
- **Rule cleanup race condition.** If a session is destroyed while DNS re-resolution is in progress, the re-resolution may re-create rules for a dead session. Document the synchronization strategy (e.g., check session state before applying DNS updates).
- **Missing integration with VM Manager for network identity.** LLD-3 references `SessionNetworkInfo` with `BridgeName`, `TapDevice`, `GuestMAC`, `GuestIP` but no LLD specifies which component assigns these values. This is either VM Manager or a bridge configuration step. Define ownership explicitly.

---

## Component 4 — Auth Proxy

- **`ProviderConfig` carries `APIKey` in memory.** The struct has `APIKey string` which is the real key. While the LLD says keys are "held in memory only," passing them through a struct that could be logged or serialized is a risk. AI agents may inadvertently log the struct. Add a `Sensitive` marker or use a `SecretRef` type that prevents accidental serialization.
- **Request count exposure conflicts with NFR-4.** `ProxyStatus.RequestCount` and `LastRequestAt` require the proxy process to expose internal state, but NFR-4 says "the proxy must not expose any shell, admin API, or general-purpose HTTP surface." Clarify how the parent process reads this data (shared memory file, signal, separate metrics channel).
- **VSOCK availability detection missing.** LLD-4 assumes vsock is available but does not specify a fallback if the host lacks vsock support. The proxy must degrade gracefully — define the fallback channel (e.g., per-session bridge IP with iptables isolation).
- **Guest environment variable injection mechanism unclear.** Section 4.3 shows environment variables injected "via cloud-init," but LLD-1 (VM Manager) owns cloud-init seed generation. Define the contract: does the proxy manager pass env vars to the VM Manager's `CloudInitData`, or does it write them to a file the VM Manager reads?
- **Concurrent request handling unspecified.** NFR-6 requires "at least 10 concurrent" requests but no concurrency model is defined (goroutine-per-request, bounded worker pool, connection limits). AI agents will default to unbounded goroutines. Specify the model.

---

## Component 5 — Shared Folder

- **`GenerateDomainXMLFragment` placement violates ARCHITECTURE.md.** This function produces VM domain XML fragments, which is VM Manager's responsibility. If shared folder defines this, it creates a conceptual dependency where VM Manager must import shared folder. Per ARCHITECTURE.md boundary rules, features must not import features. Define the XML fragment generation in VM Manager's `domain.go` with shared folder providing only mount config data.
- **Virtiofs support detection heuristic fragile.** LLD-5 suggests checking `/dev/vhost-vsock` and "libvirt virtiofs capability." `/dev/vhost-vsock` checks vsock, not virtiofs. The correct check is libvirt capability XML (`virConnectGetCapabilities`) for virtiofs support. Fix the detection logic.
- **MaxSizeMB enforcement mechanism missing.** `SharedFolderSpec.MaxSizeMB` is defined but no enforcement mechanism is specified for either virtiofs or 9p. virtiofs does not natively support size quotas. Document whether this is a best-effort hint, enforced via project quotas on the host, or deferred.

---

## Component 6 — REST API

- **Service interface redefinitions.** `SessionService`, `ProxyService`, `SharedFolderService` are redefined here but also in their respective component LLDs. AI agents implementing the REST API will not know which definitions are canonical. Reference the component LLD definitions and remove duplicates.
- **Router library choice unresolved.** The LLD says "use Go `net/http` with a router (e.g., `chi` or standard `ServeMux` with method matching)." This is an unresolved decision. For AI agents, pick one. If `chi`, add it to `go.mod` expectations. If `ServeMux` (Go 1.22+), note the minimum Go version.
- **Auth middleware scope ambiguous.** FR-9 says "require authentication on all mutating endpoints" but NFR-1's error codes include `forbidden` (403) and `not_found` (404) without specifying which endpoints return which. Provide a per-endpoint auth/authorization matrix.
- **Rate limiting deferred without tracking.** Section 7 mentions "consider adding rate limiting" and defers it. Without a tracking issue or ADR, AI agents will not revisit this. Create an ADR or add to a production-readiness checklist.
- **Request body validation rules absent.** The LLD says "all request bodies are validated before delegation" but provides no validation rules beyond "image_id non-empty, cpus > 0." Specify: max string lengths, allowed characters for owner, metadata key constraints, disk/memory upper bounds.

---

## Component 7 — Observability & Audit

- **NFR-6 "fails silently" ambiguous.** "Fails silently on logging errors" could mean audit write failures are also silent, which violates HLD's auditability requirement. Clarify: logging failures are silent, but audit write failures must be logged and retried.
- **Logger placement in `shared/logging` vs `features/`.** The Logger interface is in `shared/logging/` (a shared package) but Audit is in `features/audit/`. This asymmetry is fine but should be documented — explain why logging is shared but audit is a feature.
- **Metrics interface complexity.** The custom `Metrics`, `CounterMetric`, `GaugeMetric`, `HistogramMetric` interfaces abstract Prometheus, but the label-based API (`Inc(labels ...string)`) is non-standard for Prometheus client_golang which uses label maps. AI agents familiar with Prometheus will implement the wrong signature. Use `prometheus/client_golang` conventions directly or provide a complete adapter.
- **VM console log capture mechanism unspecified.** User Story 4.3 says "via libvirt domain console stream or serial console socket" but provides no implementation details. This is a non-trivial integration with libvirt streaming APIs. At minimum, define the `virDomainOpenConsole` or `virDomainQemuMonitorCommand` approach.
- **Audit retention purge timing unspecified.** Section 7 says "purged automatically (default: 90 days)" but does not specify when the purge runs (startup, periodic, on-write threshold). AI agents need a trigger mechanism.

---

## Component 8 — Host Resource Manager

- **Reconciliation ownership conflict.** Both LLD-2 (Session Manager) and LLD-8 (Host Resource Manager) define reconciliation logic. LLD-8's `Reconciler` interface orchestrates "full platform reconciliation" but LLD-2 has its own `ReconcileSessions`. This creates ambiguity about which component owns reconciliation. Define a single reconciliation coordinator — likely in LLD-8 as the cross-cutting concern — with LLD-2 implementing session-specific convergence.
- **`ReconcileReport` overlaps with `OrphanReport`.** LLD-1 defines `OrphanReport` for VM-level orphans and LLD-8 defines `ReconcileReport` for platform-level orphans. These serve different scopes but share structure. Unify or clearly scope: `OrphanReport` is VM-only, `ReconcileReport` is platform-wide.
- **Cgroup path derivation not documented.** The slice name `agentvm-session-<session-id>.slice` assumes session IDs are valid systemd unit names (no slashes, limited charset). If session IDs are UUIDs, this works; if they could contain special characters, document the sanitization strategy.
- **`AssignProcess` race condition.** After `VMManager.StartVM`, getting the QEMU PID and assigning it to the cgroup has a race window where the VM could fork or the PID could be recycled. Document the mitigation (e.g., assign the libvirt domain's cgroup via libvirt's `<resource>` XML element instead of PID-based assignment).
- **I/O limits implementation vague.** `IOWriteBPS` and `IOReadBPS` in `ResourceLimits` reference blkio limits, but cgroups v2 uses `io.max` with device major:minor numbers. The LLD does not specify how the overlay disk's device identifier is discovered. Define the device lookup strategy.
