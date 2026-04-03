# Package wiring

## Purpose

Configuration loading, validation, and component assembly for the AgentVM control plane. This is the **only** package permitted to import multiple feature packages (see ARCHITECTURE.md). It loads typed, validated settings from JSON files and environment variables, applies safe defaults for non-sensitive fields, and enforces fail-closed semantics for security-sensitive values.

## Exported Entrypoints

| Symbol | Description |
|--------|-------------|
| `Load(source string) (*Config, error)` | Primary entrypoint. Loads config from a JSON file path (or env-only if empty), applies defaults, validates, and returns a `*Config`. |
| `Config` | Top-level configuration struct containing `Host`, `Paths`, `API`, `Security`, `Limits`, `LogLevel`, and `SkipHostChecks` fields. |
| `ValidationError` | Structured error type with `Field`, `Value`, and `Reason` fields. |
| `ValidationErrors` | Slice of `ValidationError` with a human-readable `Error()` method. |

## Dependencies

- **Standard library only**: `encoding/json`, `fmt`, `net`, `os`, `path/filepath`, `regexp`, `strconv`, `strings`, `time`
- No external dependencies — ensures < 100ms parse time and zero dependency risk.

## Configuration Sources

Config is loaded with the following precedence (later overrides earlier):

1. Explicit defaults (safe, non-sensitive fields only)
2. JSON config file (if `source` is non-empty)
3. Environment variables (prefix: `AGENTVM_`)

### Environment Variable Mapping

| Env Variable | Config Field |
|-------------|--------------|
| `AGENTVM_HOST_LIBVIRT_URI` | `Host.LibvirtURI` |
| `AGENTVM_HOST_LIBVIRT_SOCKET_PATH` | `Host.LibvirtSocketPath` |
| `AGENTVM_HOST_QEMU_IMG_PATH` | `Host.QemuImgPath` |
| `AGENTVM_HOST_CLOUD_LOCAL_GEN_ISO_PATH` | `Host.CloudLocalGenISOPath` |
| `AGENTVM_PATHS_BASE_IMAGES_DIR` | `Paths.BaseImagesDir` |
| `AGENTVM_PATHS_OVERLAY_DISKS_DIR` | `Paths.OverlayDisksDir` |
| `AGENTVM_PATHS_CLOUD_INIT_DIR` | `Paths.CloudInitDir` |
| `AGENTVM_PATHS_DATA_DIR` | `Paths.DataDir` |
| `AGENTVM_PATHS_SQLITE_DB_PATH` | `Paths.SQLiteDBPath` |
| `AGENTVM_API_LISTEN_ADDR` | `API.ListenAddr` |
| `AGENTVM_API_READ_TIMEOUT` | `API.ReadTimeout` |
| `AGENTVM_API_WRITE_TIMEOUT` | `API.WriteTimeout` |
| `AGENTVM_API_SHUTDOWN_TIMEOUT` | `API.ShutdownTimeout` |
| `AGENTVM_SECURITY_ADMIN_TOKEN` | `Security.AdminToken` |
| `AGENTVM_SECURITY_TLS_CERT_PATH` | `Security.TLSCertPath` |
| `AGENTVM_SECURITY_TLS_KEY_PATH` | `Security.TLSKeyPath` |
| `AGENTVM_SECURITY_TLS_ENABLED` | `Security.TLSEnabled` |
| `AGENTVM_LIMITS_MAX_CONCURRENT_VMS` | `Limits.MaxConcurrentVMs` |
| `AGENTVM_LIMITS_MAX_CONCURRENT_SESSIONS` | `Limits.MaxConcurrentSessions` |
| `AGENTVM_LIMITS_VM_START_TIMEOUT` | `Limits.VMStartTimeout` |
| `AGENTVM_LIMITS_DISK_SIZE_DEFAULT_GB` | `Limits.DiskSizeDefaultGB` |
| `AGENTVM_LOG_LEVEL` | `LogLevel` |
| `AGENTVM_SKIP_HOST_CHECKS` | `SkipHostChecks` |

## Notes

- **Fail-closed**: `Security.AdminToken` has **no default**. An empty or short (< 32 chars) token causes a hard validation failure.
- **No silent fallbacks**: All validation errors are aggregated and returned together — the config is never partially valid.
- **Host prerequisites**: Binary paths, socket paths, and data directories are verified at load time. Set `AGENTVM_SKIP_HOST_CHECKS=true` to bypass in test environments.
- **Performance**: Config parsing completes in < 100ms (typically ~15-20ms).
