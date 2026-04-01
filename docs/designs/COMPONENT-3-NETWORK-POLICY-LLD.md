# Component 3 — Network Policy Engine

**Purpose**: Enforce per-session outbound network policy (iptables/nftables), host isolation, block private ranges.

## Functional Requirements

| ID | Requirement |
|----|-------------|
| FR-1 | Apply policy at creation |
| FR-2 | Support strict/restricted/permissive modes |
| FR-3 | Enforce host isolation (VM→host blocked) |
| FR-4 | Allow/block at IP/CIDR and DNS-name granularity |
| FR-5 | Runtime policy mutation |
| FR-6 | Remove rules on destroy |
| FR-7 | Generate iptables/nftables rules |
| FR-8 | Reconcile on startup |
| FR-9 | Emit audit events |
| FR-10 | Fail closed |

## Non-Functional Requirements

| ID | Requirement |
|----|-------------|
| NFR-1 | Policy application < 5s |
| NFR-2 | Deterministic rule generation |
| NFR-3 | No state beyond SQLite |
| NFR-4 | DNS: resolve on apply, re-resolve periodically |
| NFR-5 | Cleanup idempotent |
| NFR-6 | Fail closed |

## Contracts

```go
type NetworkPolicyApplier interface {
    ApplyPolicy(ctx context.Context, sessionID SessionID, policy NetworkPolicy) error
    RemovePolicy(sessionID SessionID) error
    GetPolicy(sessionID SessionID) (NetworkPolicy, error)
    AllowDestination(sessionID SessionID, dest Destination) error
    BlockDestination(sessionID SessionID, dest Destination) error
    ResetPolicy(sessionID SessionID) error
}
```

**FirewallBackend** (in backend.go, interface separate from impl):
```go
type FirewallBackend interface {
    ApplyRules(chainName string, rules []FirewallRule) error
    FlushChain(chainName string) error
    ListChains(prefix string) ([]string, error)
}
```

## Implementation Notes

- **DNS Resolution**: ADR - use 5-min fixed interval, TTL tracking deferred
- **Rule Ordering**: Strict mode - private-range DROP before default DROP
- **Cleanup Sync**: Check session exists before applying DNS update
- **Network Identity**: VM Manager assigns BridgeName/TapDevice/GuestMAC/GuestIP
- **Types**: NetworkPolicyMode, NetworkPolicy, Destination in shared/types/network.go