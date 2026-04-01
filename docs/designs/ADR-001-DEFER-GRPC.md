# ADR 001: Defer Internal gRPC Contracts

## Status

Deferred

## Context

HLD Section 16.2 and Section 19 require internal gRPC contracts for future extensibility. However, initial deployment uses single-process in-process communication behind Go interfaces.

## Decision

Defer gRPC contract definition until Phase 6 or when remote deployment is required.

### Rationale

1. **YAGNI**: Single-host deployment doesn't require network boundaries between components
2. **Complexity**: gRPC adds generated code, protobufs, service registration
3. **Flexibility preserved**: Interface design allows future gRPC wrapping without breaking contracts
4. **HLD compatible**: HLD allows "single-binary deployment may still run most components in-process

### Future Work

When needed:
1. Define `.proto` files for each feature boundary
2. Generate Go gRPC servers from existing interfaces
3. Add gRPC reflection for debugging

## References

- HLD Section 16.2: Internal gRPC contracts
- HLD Section 19: Technology Direction - Internal contracts
- ARCHITECTURE.md: wiring/ is sole connection point