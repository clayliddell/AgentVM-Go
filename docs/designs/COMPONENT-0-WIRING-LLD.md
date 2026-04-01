# Component 0 — Wiring Layer

## 1. Purpose

The wiring layer is the **sole package that connects features** per ARCHITECTURE.md. It assembles all feature dependencies, wires consumer interfaces to provider implementations, and starts the control plane services.

## 1.1 Functional Requirements

The wiring layer must:

1. Assemble all feature services with their dependencies during startup
2. Run SQLite migrations against the configured database path
3. Produce a configured HTTP server with all handlers wired
4. Verify feature dependencies satisfy required interfaces before starting
5. Handle partial assembly failures gracefully with deterministic rollback

## 1.2 Non-Functional Requirements

The wiring layer must:

1. **Reliability**: If wiring fails, report error and prevent server start; no partial state
2. **Testability**: Assembly logic must be testable with mock dependencies
3. **Determinism**: Same config always produces same dependency graph
4. **Security**: No secret values logged during assembly; Sensitive types handled correctly
5. **Performance**: Assembly completes in <1s for typical dependency count
6. **Observability**: Log all wiring steps at appropriate level

## 2. Structure

```
internal/
  wiring/
    server.go      # HTTP server assembly + feature wiring
    config.go      # configuration loading
    assembly.go    # interface wiring logic
```

## 3. Wiring Assembly

### 3.1 Interface Wiring Pattern

Per ARCHITECTURE.md, consuming features define interfaces; providing features satisfy them implicitly. The wiring layer connects concrete instances:

```go
// wiring/assembly.go

func Assemble(cfg *Config) (*Server, error) {
    // Shared dependencies
    logger := logging.NewLogger(cfg.LogLevel)
    metrics := metrics.NewRegistry()
    db := sqlite.Open(cfg.DBPath)
    migrator := migrate.NewMigrator(db)
    
    // Run migrations
    if err := migrator.Run(cfg.MigrationsPath); err != nil {
        return nil, fmt.Errorf("migration failed: %w", err)
    }
    
    // Feature instantiation
    imageStore := image.NewStore(db, logger)
    imageSvc := image.NewService(imageStore, logger)
    
    vmStore := vm.NewStore(db, logger)
    vmLifecycle := vm.NewLifecycle(vmStore, logger)
    libvirtClient := libvirt.NewClient() // satisfies LibvirtClient
    vmSvc := vm.NewService(vmLifecycle, libvirtClient)
    
    sessionStore := session.NewStore(db, logger)
    sessionSvc := session.NewService(sessionStore, vmSvc, networkSvc, proxySvc, sharedFolderSvc, logger)
    
    networkBackend := firewall.NewNftablesBackend(logger)
    networkSvc := network.NewService(networkBackend, sessionStore, logger)
    
    proxySvc := proxy.NewService(cfg.ProxyConfig, logger)
    sharedFolderSvc := sharedfolder.NewService(cfg.SharedFolderBasePath, logger)
    
    // Observability
    auditStore := audit.NewStore(db, logger)
    auditEmitter := audit.NewEmitter(auditStore, logger)
    healthSvc := health.NewService(vmSvc, proxySvc, networkSvc, sharedFolderSvc, logger)
    
    // Reconciliation coordinator (owns cross-feature reconciliation)
    reconciler := resources.NewReconciler(vmSvc, proxySvc, networkSvc, sharedFolderSvc, sessionStore, logger)
    
    return &Server{
        SessionService: sessionSvc,
        ImageService:  imageSvc,
        NetworkService: networkSvc,
        ProxyService:  proxySvc,
        SharedFolderService: sharedFolderSvc,
        AuditService:  audit.NewService(auditStore, logger),
        HealthService: healthSvc,
        Logger:        logger,
        Metrics:       metrics,
    }, nil
}
```

## 4. SQLite Migration Infrastructure

### 4.1 Migration Directory Structure

```
migrations/
  001_create_sessions.sql
  002_create_images.sql
  003_create_audit_events.sql
  004_create_network_policies.sql
```

### 4.2 Version Numbering

Migrations numbered sequentially (001, 002, ...). Each migration is idempotent.

### 4.3 Migration Runner

```go
// internal/shared/migrate/migrator.go

type Migrator interface {
    Run(path string) error
}

type migration struct {
    Version int
    SQL     string
}

func (m *Migrator) Run(path string) error {
    // Read migration files, sort by version
    // Apply each migration if not already applied
    // Track applied versions in db
}
```

## 5. Rollback Failure Handling

When rollback itself partially fails:

1. Log all rollback errors with session ID
2. Mark session as `error` with detailed message
3. On next startup, reconciliation detects inconsistency
4. Reconciliation performs cleanup: destroy VM, remove proxy, clear network rules
5. Session converges to `destroyed`

Per HLD NFR: "deterministic cleanup and reconciliation after crashes."

## 6. Concurrency Serialization

Session creation serialized using `sync.Mutex` per session ID:

```go
type SessionMutexes struct {
    mu    sync.Mutex
    locks map[SessionID]*sync.Mutex
}

func (s *SessionMutexes) Acquire(id SessionID) func() {
    s.mu.Lock()
    if s.locks[id] == nil {
        s.locks[id] = &sync.Mutex{}
    }
    m := s.locks[id]
    s.mu.Unlock()
    m.Lock()
    return m.Unlock
}
```

## 7. Canonical Types Reference

Types defined once in `shared/types/` and referenced by all LLDs:

| Type | Location | Notes |
|------|----------|-------|
| SessionID, Session, SessionStatus | `shared/types/session.go` | All components reference |
| ImageID, ImageInfo | `shared/types/image.go` | VM Manager defines impl |
| NetworkPolicy, Destination | `shared/types/network.go` | Network engine defines impl |
| ConnectionInfo | `shared/types/vm.go` | Renamed from SessionConnectionInfo |
| SharedFolderSpec | `shared/types/sharedfolder.go` | Shared folder defines impl |
| ProxyConfig | `shared/types/proxy.go` | Proxy defines impl |
| AuditEvent | `shared/types/audit.go` | Audit feature defines impl |
| ResourceLimits | `shared/types/resources.go` | Resources defines impl |

All LLDs mark types as "defined in `shared/types/XYZ`, shown here for reference only."