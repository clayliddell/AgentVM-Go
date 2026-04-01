# Phase 4: Network Isolation

## Tasks

### Network Policy Engine (Component 3)
- Implement policy application at session creation
- Implement strict/restricted/permissive modes
- Implement host isolation rules (VM→host blocked)
- Implement allow/block at IP/CIDR granularity
- Implement DNS-name based policy
- Implement runtime policy mutation (allow/block/reset)
- Implement rule cleanup on destroy
- Implement iptables/nftables rule generation
- Implement network reconciliation on startup
- Separate FirewallBackend interface

## Functional Requirements

1. Network policy is applied automatically during session creation before VM starts.
2. Strict mode: all egress denied by default; only explicitly allowed destinations reachable.
3. Restricted mode: general internet egress allowed; operator can block specific destinations.
4. Permissive mode: broad egress allowed; VM→host and VM→VM still blocked.
5. Host isolation: VM cannot reach host IP, host services, or other VM IPs under any mode.
6. Allow/block rules support IP and CIDR granularity (e.g., 1.2.3.4, 10.0.0.0/8).
7. DNS-name based policy: rules can target domain names; resolved IPs enforced dynamically.
8. Runtime mutation: allow/block/reset API calls update policy without session restart.
9. On session destroy, all firewall rules for that session are removed.
10. iptables and nftables backends supported behind a FirewallBackend interface.
11. On startup, reconciliation detects stale firewall rules and cleans up orphans.

## Non-Functional Requirements

- Firewall rule application < 500ms per session.
- Runtime policy mutation takes effect within 2s.
- Rule cleanup on destroy is atomic (all-or-nothing per session).
- FirewallBackend interface is swappable without changing policy logic.
- DNS policy resolution does not block rule application (async with initial deny).
- Reconciliation on startup completes in < 10s regardless of rule count.

## E2E Test Conditions

1. Create a session with strict policy; verify no egress except allowed destinations.
2. Create a session with restricted policy; verify general internet egress works.
3. Create a session with permissive policy; verify broad egress works.
4. From guest, attempt to reach host IP; verify blocked in all modes.
5. From one guest, attempt to reach another guest IP; verify blocked.
6. Add an allow rule for a specific IP; verify guest can reach it.
7. Add a block rule for a CIDR; verify guest cannot reach that range.
8. Add a DNS-based allow rule; verify guest can reach the domain after resolution.
9. Call runtime mutation: allow a new destination; verify guest can reach it within 2s.
10. Call runtime mutation: block a destination; verify guest cannot reach it.
11. Call runtime reset; verify policy reverts to default for the mode.
12. Destroy session; verify all iptables/nftables rules for that session removed.
13. Restart service with existing sessions; verify reconciliation detects no stale rules.
14. Introduce stale rules manually; restart; verify reconciliation cleans them.
15. Test with iptables backend; verify rules generated correctly.
16. Test with nftables backend; verify rules generated correctly.
17. Run go vet and project linters; zero violations.
