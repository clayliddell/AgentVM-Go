# AgentVM Platform — High-Level Design

## 1. Purpose

AgentVM is a secure VM orchestration platform for running AI agents inside isolated, sandboxed virtual machines on a single host. It is designed for adversarial or semi-trusted agent workloads that may need:

* full Linux environments,
* root inside the guest,
* nested virtualization,
* controlled internet access,
* access to project files through a shared folder,
* access to external AI APIs without exposing real credentials inside the VM.

The platform prioritizes:

1. **Host security**
2. **Strong isolation between sessions**
3. **Operational simplicity**
4. **A clean contract for external orchestrators**

---

## 2. Scope

### In scope

* Single-host control plane for VM-backed agent sessions
* Secure lifecycle management for sessions
* Host-enforced network isolation and policy
* Host-side secret mediation via auth proxy
* Shared folder between host and guest
* Session metadata, audit, metrics, and health
* External API for operators and orchestrators
* Nested virtualization support

### Out of scope

* Multi-host scheduling
* Live migration
* High-density container execution
* Full cluster management
* GPU support in the initial version
* Guest-level policy enforcement as the primary security boundary

---

## 3. Key Design Decisions

### 3.1 Isolation model

Use **full VMs**, not containers, as the primary isolation boundary because agents are assumed capable of hostile or unsafe behavior.

### 3.2 Hypervisor stack

Use **KVM/QEMU managed through libvirt**.

This is the only practical option in this design that simultaneously provides:

* production-grade hardware isolation,
* mature operational tooling,
* nested virtualization,
* compatibility with standard Linux guests.

### 3.3 Implementation platform

The platform is implemented primarily in **Go**.

Go is used for:

* control plane services,
* API layers,
* lifecycle orchestration,
* metadata management,
* audit and observability,
* auth proxy.

The platform may rely on **existing system tooling** where appropriate, including:

* libvirt/libvirtd,
* QEMU/KVM,
* cloud-init,
* Linux firewall tooling,
* cgroups v2,
* SELinux/sVirt,
* virtiofs or 9p,
* systemd.

### 3.4 Metadata store

Use **SQLite** for single-host metadata and audit indexing.

Rationale:

* zero external dependency,
* operationally simple,
* sufficient for initial scale,
* easy to replace later if needed.

### 3.5 API strategy

Expose:

* **REST/JSON externally** for operators and orchestrators,
* **gRPC internally** for service contracts and future extensibility.

A single-binary deployment may still run most components in-process behind Go interfaces.

### 3.6 Secret handling

Real provider API keys must **never enter the guest VM**.
All provider access goes through a **host-side auth proxy**.

### 3.7 Network model

Default to **deny by policy** and expose only the minimum egress needed for the session.

### 3.8 Shared filesystem model

Use **virtiofs** as the preferred shared-folder mechanism, with **9p** as fallback.

---

## 4. Goals and Quality Attributes

## 4.1 Primary goals

* Safely run AI agents in isolated VMs
* Support workloads that require nested virtualization
* Prevent cross-session access
* Prevent guest access to host secrets
* Provide a simple control plane for session creation, control, and cleanup
* Provide a stable integration contract for external orchestrators

## 4.2 Quality attributes

* **Security:** defense in depth; no single control should be solely trusted
* **Correctness:** cleanup must be reliable even after crashes
* **Auditability:** lifecycle, network, and secret access must be traceable
* **Simplicity:** minimize moving parts in the first release
* **Extensibility:** preserve room for future backends and multi-node evolution

---

## 5. Functional Requirements

The platform must:

1. Create, start, stop, destroy, and inspect agent sessions.
2. Provision each session as a dedicated VM from a managed base image.
3. Support base images with capability metadata, including nested virtualization requirements.
4. Inject SSH access for operators or orchestrators.
5. Provide a host↔guest shared folder.
6. Provide outbound network control per session.
7. Prevent VM→host and VM→VM communication by default.
8. Mediate external API access through a host-side auth proxy.
9. Support session metadata and ownership tracking.
10. Record audit events for lifecycle, policy changes, and proxy use.
11. Surface health, logs, and metrics per session.
12. Expose an external API suitable for automation by a separate orchestrator.
13. Clean up disks, network rules, proxy state, and shared-folder state when a session ends.
14. Support nested virtualization for guests that require it.

---

## 6. Non-Functional Requirements

The platform must:

* treat the guest as potentially adversarial,
* preserve host integrity if a guest is compromised,
* preserve session isolation if another session is compromised,
* ensure real secrets remain outside the guest trust boundary,
* enforce CPU, memory, disk, and process limits at the host level,
* fail closed on policy or provisioning errors,
* provide deterministic cleanup and reconciliation after crashes,
* remain operable by a small team on a single host,
* support low session density rather than optimize for maximum density.

---

## 7. System Context

AgentVM runs on a Linux host with KVM support enabled. An operator or an external orchestrator calls the public API to create a session. AgentVM provisions a VM from a base image, attaches a shared folder, configures network isolation, starts a per-session auth proxy, and returns connection details. The agent executes inside the guest. The host remains the authority for security, policy, secrets, and auditing.

---

## 8. High-Level Architecture

```text
External Orchestrator / Operators
            |
        REST API
            |
      AgentVM Control Plane (Go)
            |
   +--------+--------+---------+---------+---------+
   |                 |         |         |         |
Session Manager   Image/Store  Network   Proxy    Observe/Audit
   |                           Policy    Manager
   +-----------------+---------+---------+---------+
                     |
                 VM Manager
                     |
                 libvirt/libvirtd
                     |
                 QEMU/KVM Guest
                     |
        +------------+-------------+
        |                          |
   Shared Folder               Restricted Network
  (virtiofs / 9p)             (host-enforced)
```

### Main subsystems

* **REST API:** external control surface
* **gRPC/internal contracts:** internal service boundaries and future remote control
* **Session Manager:** lifecycle orchestration and state transitions
* **VM Manager:** image selection, domain creation, boot, destroy, reconciliation
* **Network Policy Engine:** session egress policy and host isolation rules
* **Auth Proxy Manager:** secret mediation and provider request logging
* **Storage/Metadata Manager:** images, overlays, shared folders, SQLite state
* **Observability/Audit:** logs, metrics, health, audit events

---

## 9. Core Domain Model

## 9.1 Session

A **session** is the primary workload abstraction.

A session includes:

* session ID
* owner
* status
* base image
* resource allocation
* network policy
* SSH info
* auth proxy info
* shared-folder info
* metadata/tags
* health state

### Session states

* requested
* creating
* running
* stopping
* destroyed
* error

### Session state machine

Normal flow is **requested → creating → running → stopping → destroyed**. Failures during provisioning or runtime move the session to **error**, after which the control plane may either retry from a safe point or force cleanup to **destroyed**; on restart or reconciliation, observed host reality is authoritative and any ambiguous in-progress state must converge to **running**, **error**, or **destroyed**, never remain indefinitely in **creating** or **stopping**.

The external API and orchestrator contract should operate on **sessions**, not raw VMs.

## 9.2 Image

A base image defines the guest environment and capability hints, such as:

* OS and version
* architecture
* requires nested virtualization
* supports Docker/Podman
* minimum resource requirements

## 9.3 Network policy

Each session has one of three outbound policies:

* **strict**: deny by default; allow explicit destinations only
* **restricted**: allow internet egress except blocked destinations
* **permissive**: broad internet egress, still no host/private-network lateral access

## 9.4 Shared folder

Each session gets a dedicated shared directory on the host, mounted in the guest at a fixed location. The guest must not be able to escape this boundary.

## 9.5 Auth proxy

Each session gets a dedicated auth proxy context. The guest receives only dummy credentials and a local provider base URL.

---

## 10. Security Architecture

## 10.1 Threat model

The guest agent may:

* run arbitrary code,
* install tools,
* attempt host discovery,
* attempt lateral movement,
* attempt credential theft,
* attempt resource exhaustion,
* attempt persistence or data exfiltration.

The design assumes guest compromise is possible and still requires the host and other sessions to remain protected.

## 10.2 Security invariant

**No single control is trusted as the only containment layer.**
If one layer fails, remaining layers must still prevent VM escape, cross-session access, or host secret exposure.

## 10.3 Security layers

### Layer 1: Hardware virtualization

KVM provides hardware-backed separation of guest memory and execution.

### Layer 2: Host network isolation

The host blocks:

* VM→host access,
* VM→VM access,
* access to private/internal address ranges unless explicitly allowed.

### Layer 3: Resource isolation

cgroups v2 enforce:

* CPU limits,
* memory limits,
* process limits,
* I/O limits.

### Layer 4: Mandatory access control

SELinux with sVirt constrains VM processes and storage access on the host.

### Layer 5: Minimal VM device model

Only required virtual devices are exposed. Unneeded devices remain disabled to reduce attack surface.

### Layer 6: Guest hardening

Guests use minimal images and predictable configuration. Guest policy is defense in depth, not the primary security boundary.

### Layer 7: Host-side secret mediation

Real API keys remain on the host. The guest can only reach providers through the auth proxy using dummy credentials.

---

## 11. Shared Folder Security Model

The shared folder is the only intended file exchange path between host and guest.

Requirements:

* each session gets a unique host directory,
* the guest sees only that directory,
* symlink and path traversal escapes must be prevented,
* the guest must not gain visibility into arbitrary host paths,
* host operators can inspect session output,
* mount behavior must be deterministic and auditable.

Preferred implementation:

* **virtiofs**
  Fallback:
* **9p**

**virtiofs** should be used whenever host and guest support it because it provides the preferred performance and isolation characteristics for this platform; **9p** is a compatibility fallback only when virtiofs is unavailable or unsupported for the selected image/host combination.

The HLD requires boundary enforcement and auditability, but does not mandate a specific low-level mount implementation beyond these supported mechanisms.

---

## 12. Auth Proxy Security Model

The auth proxy is a host-side control that:

* receives requests from the guest,
* validates session identity,
* replaces dummy credentials with real provider credentials,
* forwards requests upstream,
* logs metadata about requests,
* never exposes real secrets to the guest.

### Required properties

* real keys stored only on the host,
* per-session isolation of proxy context,
* source validation tied to session identity,
* no shell or general-purpose admin surface,
* minimal runtime privileges,
* auditable request flow.

Guest-to-proxy transport must use a host-controlled per-session channel, preferably **virtio-vsock** or an equivalently isolated host↔guest path, and must not rely on a shared cross-session listener. Session identity must be bound to both the control-plane session ID and the originating VM/channel identity so possession of dummy credentials alone is insufficient to impersonate another session.

A separate hardened Go process per session is acceptable and preferred for stronger isolation.

---

## 13. Networking Model

## 13.1 Principles

* default deny where practical,
* no inbound guest reachability from untrusted networks,
* no VM→host lateral access,
* no VM→VM connectivity,
* internet egress is policy-controlled per session.

## 13.2 Policy modes

### Strict

Only explicit destinations are reachable.

### Restricted

General egress allowed, but operator-controlled blocks can be applied.

### Permissive

Broad egress allowed for compatible workflows, while still preserving host and private-network isolation.

Allowed or blocked destinations must support policy expression at least at the granularity of **IP/CIDR and DNS name**, with optional finer transport details deferred to the LLD.

## 13.3 Runtime policy

The platform should support runtime policy mutation through the public API:

* allow destination
* block destination
* reset policy

The specific firewall implementation may vary by Linux distribution and may use existing firewall tooling.

---

## 14. VM Lifecycle

### Create

1. Validate request and capacity
2. Select image
3. Create overlay disk from base image
4. Prepare cloud-init or equivalent bootstrap data
5. Allocate network identity and policy
6. Create shared-folder mount
7. Create auth proxy context
8. Define and start VM through libvirt
9. Wait for session reachability
10. Persist state and emit audit events

### Destroy

1. Stop guest if possible
2. Force destroy if required
3. Remove VM definition
4. Remove overlay and runtime artifacts
5. Remove network policy state
6. Stop auth proxy
7. Clean up shared-folder runtime state
8. Persist terminal state and emit audit events

### Reconcile

On service restart, AgentVM must reconcile:

* orphaned libvirt domains,
* stale proxy state,
* stale network policy state,
* incomplete session records.

---

## 15. Data and Storage Model

### Managed host storage

A single base directory contains:

* base images
* per-session overlays
* shared directories
* proxy configuration/state
* logs
* SQLite metadata

### Storage requirements

* base images are read-only and versioned
* runtime disks are disposable overlays
* session storage is isolated by session ID
* metadata must survive service restarts
* cleanup must remove transient state without corrupting image sources

Reconciliation must prefer **surviving and re-attaching valid in-use session artifacts** after control-plane restart, and only destroy artifacts proven orphaned or inconsistent with authoritative session state.

---

## 16. Interfaces

## 16.1 External REST API

The public API should expose resource-oriented endpoints for:

* sessions
* images
* network policy
* proxy status
* shared-folder status
* logs and audit
* host health and capacity
* backend capabilities

REST is the automation surface for users and orchestrators. The public API must require authenticated callers and enforce authorization on session-scoped operations, with all mutating requests attributable to an authenticated principal and recorded in audit logs.

## 16.2 Internal gRPC contracts

Internal contracts should define service boundaries for:

* session operations
* VM operations
* network policy
* image catalog
* observability

In the first release these services may run in-process. The gRPC contract exists to preserve future flexibility.

---

## 17. External Orchestrator Integration Contract

AgentVM should expose a backend contract that allows an external orchestrator to:

* create a session,
* destroy a session,
* query status,
* retrieve connection info,
* inspect capabilities,
* mutate network policy,
* supply provider credentials through approved channels.

The orchestrator contract must remain:

* backend-neutral,
* capability-aware,
* future-proof for additional isolation backends.

The contract should focus on **session semantics**, not hypervisor-specific implementation details.

---

## 18. Observability and Audit

The platform must provide:

### Metrics

* session counts
* resource consumption by session
* host capacity
* proxy activity
* error rates
* boot and destroy timings

### Logs

* control-plane logs
* VM console logs
* proxy request logs
* security-relevant events

### Audit events

At minimum:

* session requested/created/running/destroyed/error
* image selected
* network policy changed
* proxy started/stopped/requested
* shared folder mounted/unmounted
* resource limit reached
* reconciliation action taken

Audit logs must be host-owned and inaccessible to guests.

---

## 19. Technology Direction

| Area               | Choice                                          |
| ------------------ | ----------------------------------------------- |
| Main platform      | Go                                              |
| Hypervisor         | KVM/QEMU                                        |
| VM management      | libvirt                                         |
| Public API         | REST/JSON                                       |
| Internal contracts | gRPC                                            |
| Metadata           | SQLite                                          |
| Resource control   | cgroups v2                                      |
| Host MAC           | SELinux/sVirt                                   |
| Shared folder      | virtiofs preferred, 9p fallback                 |
| Secret mediation   | host-side auth proxy                            |
| Guest bootstrap    | cloud-init or equivalent                        |
| Logging/metrics    | structured logs + Prometheus-compatible metrics |

---

## 20. Host Requirements

The host must provide:

* Linux with KVM support
* nested virtualization enabled
* libvirt and QEMU installed
* cgroups v2
* SELinux enforcing
* firewall support sufficient for per-session egress control
* systemd or equivalent service management
* local disk suitable for base images and overlays

The host must be treated as a hardened appliance, not a general-purpose multi-user system.

---

## 21. Delivery Phases

### Phase 1: Secure VM lifecycle

* image management
* VM create/destroy
* SSH reachability
* SQLite metadata
* basic audit

### Phase 2: Session model

* session abstraction
* shared folder
* auth proxy
* health checks

### Phase 3: Public control plane

* REST API
* basic CLI
* capability reporting
* session inspection

### Phase 4: Network isolation

* strict/restricted/permissive modes
* runtime allow/block/reset
* host/private-network protection

### Phase 5: Resource enforcement and reconciliation

* cgroups v2 enforcement
* crash recovery
* orphan cleanup
* capacity management

### Phase 6: Security and production readiness

* SELinux/sVirt validation
* red-team tests
* metrics and dashboards
* operator hardening guidance

---

## 22. Risks and Mitigations

| Risk                                          | Mitigation                                                             |
| --------------------------------------------- | ---------------------------------------------------------------------- |
| VM escape via hypervisor/device vulnerability | minimal device model, patching, multiple containment layers            |
| Network misconfiguration                      | generated policy only, integration tests, fail-closed behavior         |
| Resource exhaustion                           | admission control, cgroups, host reservation                           |
| Shared-folder boundary escape                 | dedicated per-session mounts, traversal protections, adversarial tests |
| Secret leakage                                | host-only real credentials, hardened proxy, audit                      |
| Crash leaves orphaned resources               | reconciliation on startup, terminal cleanup logic                      |
| Over-specification too early                  | keep HLD focused on contracts and invariants, defer mechanisms to LLD  |
| Future orchestrator changes                   | session-centric contract with explicit capability discovery            |

---

## 23. Summary

AgentVM is a **Go-first, single-host VM isolation platform** for AI agents that require stronger boundaries than containers can provide. Its architecture is centered on a **session abstraction**, **KVM/libvirt-based VM lifecycle**, **host-enforced network and resource controls**, **per-session shared folders**, and a **host-side auth proxy that keeps real secrets out of guests**.

The defining design choice is simple: **treat the guest as hostile and keep the host in control of policy, secrets, and cleanup**. That choice drives the platform’s use of full VMs, defense-in-depth security, low operational complexity, and a clean external integration contract for orchestrators.
