# Phase 6: Security & Production Readiness

## Tasks

### Security Hardening
- Validate SELinux/sVirt configuration
- Implement red-team test plan
- Document operator hardening guidance
- Implement rate limiting (ADR)
- Implement HTTPS configuration

### Final Integration
- End-to-end session creation/destroy flow
- Crash recovery and reconciliation testing
- Performance benchmarking (latency, throughput)
- Security penetration testing
- Metrics dashboards setup
- Operational runbook documentation

## Functional Requirements

1. SELinux/sVirt labels applied correctly to VM processes and disk images.
2. Red-team test plan covers: VM escape, cross-session access, credential theft, resource exhaustion.
3. Operator hardening guide documents: host setup, kernel params, firewall baseline, SELinux policy.
4. Rate limiting enforced on API endpoints (configurable limits per endpoint/route).
5. HTTPS supported with configurable TLS cert/key paths; HTTP redirects to HTTPS.
6. Full E2E flow: API call → session create → agent runs → session destroy → cleanup verified.
7. Crash recovery: kill service at each lifecycle stage; restart; verify consistent state.
8. Performance benchmarks: session create latency, concurrent session throughput, API latency.
9. Penetration testing confirms no exploitable VM escape, cross-session, or credential leakage paths.
10. Prometheus dashboards for: session count, resource usage, error rates, latency percentiles.
11. Operational runbook covers: startup, shutdown, troubleshooting, scaling limits, incident response.

## Non-Functional Requirements

- Session creation P95 latency < 120s (image-dependent).
- API P95 latency < 200ms (non-VM operations).
- Support 20 concurrent sessions on reference hardware.
- Rate limiting adds < 5ms overhead per request.
- TLS handshake < 10ms on local network.
- Zero known high/critical CVEs in dependency scan.
- Documentation covers all operational procedures with copy-paste commands.

## E2E Test Conditions

1. Verify SELinux labels on VM process (ps -Z) and disk images (ls -Z).
2. Execute red-team test plan; all attack vectors fail or are contained.
3. Follow hardening guide on fresh host; verify system meets security baseline.
4. Exceed rate limit on API; verify 429 Too Many Requests returned.
5. Connect to API via HTTP; verify redirect to HTTPS.
6. Connect to API via HTTPS; verify valid TLS and successful request.
7. Full E2E: POST session → VM boots → agent executes task → DELETE session → all resources cleaned.
8. Kill service during VM creation; restart; verify reconciliation converges to correct state.
9. Kill service during session destroy; restart; verify cleanup completes.
10. Run benchmark suite; verify latency and throughput meet targets.
11. Run penetration tests; verify no exploitable findings.
12. Scrape /metrics; verify all dashboard metrics populated.
13. Follow runbook for common incidents; verify procedures work as documented.
14. Run go vet and project linters; zero violations.
