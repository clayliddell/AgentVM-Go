// Package wiring provides configuration loading, validation, and component
// assembly for the AgentVM control plane. It is the only package permitted
// to import multiple feature packages (see ARCHITECTURE.md).
//
// Configuration is loaded from a JSON file and/or environment variables
// (prefix: AGENTVM_). Environment variables take precedence over file values.
// Security-sensitive fields have no defaults and cause validation to fail
// if not explicitly set (fail-closed semantics).
package wiring

import (
	"encoding/json"
	"fmt"
	"os"

	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Top-level Config
// ---------------------------------------------------------------------------

// Config holds all configuration for the AgentVM control plane.
type Config struct {
	Host           HostConfig
	Paths          PathConfig
	API            APIConfig
	Security       SecurityConfig
	Limits         LimitsConfig
	LogLevel       string
	SkipHostChecks bool
}

// ---------------------------------------------------------------------------
// Sub-structs
// ---------------------------------------------------------------------------

// HostConfig holds host-level prerequisites and paths.
type HostConfig struct {
	LibvirtURI           string
	LibvirtSocketPath    string
	QemuImgPath          string
	CloudLocalGenISOPath string
}

// PathConfig holds filesystem paths used by the control plane.
type PathConfig struct {
	BaseImagesDir   string
	OverlayDisksDir string
	CloudInitDir    string
	DataDir         string
	SQLiteDBPath    string
}

// APIConfig holds HTTP server settings.
type APIConfig struct {
	ListenAddr      string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// SecurityConfig holds security-sensitive settings.
type SecurityConfig struct {
	AdminToken  string
	TLSCertPath string
	TLSKeyPath  string
	TLSEnabled  bool
}

// LimitsConfig holds operational limits.
type LimitsConfig struct {
	MaxConcurrentVMs      int
	MaxConcurrentSessions int
	VMStartTimeout        time.Duration
	DiskSizeDefaultGB     int
}

// ---------------------------------------------------------------------------
// ValidationError types
// ---------------------------------------------------------------------------

// ValidationError describes a single configuration validation failure.
type ValidationError struct {
	Field  string
	Value  string
	Reason string
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []ValidationError

// Is implements errors.Is for ValidationErrors.
func (ve ValidationErrors) Is(target error) bool {
	_, ok := target.(ValidationErrors)
	return ok
}

// Error returns a human-readable multi-line error message.
func (ve ValidationErrors) Error() string {
	count := len(ve)
	if count == 0 {
		return ""
	}

	var b strings.Builder
	if count == 1 {
		b.WriteString("configuration validation failed: ")
	} else {
		b.WriteString("configuration validation failed (")
		b.WriteString(strconv.Itoa(count))
		b.WriteString(" errors): ")
	}
	b.WriteString("\n")
	for _, e := range ve {
		b.WriteString("  - ")
		b.WriteString(e.Field)
		b.WriteString(": ")
		b.WriteString(e.Reason)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// ---------------------------------------------------------------------------
// Valid log levels
// ---------------------------------------------------------------------------

var validLogLevels = []string{"debug", "info", "warn", "error"}

// ---------------------------------------------------------------------------
// Load reads configuration from a JSON file (if source is non-empty) and
// environment variables, applies defaults, validates, and returns the config.
// Environment variables take precedence over file values.
//
// If source is an empty string, config is loaded from environment variables only.
func Load(source string) (*Config, error) {
	cfg := &Config{}

	// Apply explicit defaults for non-sensitive fields.
	applyDefaults(cfg)

	// Load from file if a source path is provided.
	if source != "" {
		if err := loadFromFile(source, cfg); err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}

	// Override with environment variables.
	if err := loadFromEnv(cfg); err != nil {
		return nil, fmt.Errorf("failed to load config from environment: %w", err)
	}

	// Validate all settings.
	if errs := validate(cfg); len(errs) > 0 {
		return nil, errs
	}

	return cfg, nil
}

// ---------------------------------------------------------------------------
// applyDefaults sets safe defaults for all non-sensitive fields.
// Sensitive fields (e.g., AdminToken) are intentionally left at zero-value.
func applyDefaults(cfg *Config) {
	// Host defaults
	if cfg.Host.LibvirtURI == "" {
		cfg.Host.LibvirtURI = "qemu:///system"
	}
	if cfg.Host.LibvirtSocketPath == "" {
		cfg.Host.LibvirtSocketPath = "/var/run/libvirt/libvirt-sock"
	}
	if cfg.Host.QemuImgPath == "" {
		cfg.Host.QemuImgPath = "/usr/bin/qemu-img"
	}
	if cfg.Host.CloudLocalGenISOPath == "" {
		cfg.Host.CloudLocalGenISOPath = "/usr/bin/genisoimage"
	}

	// Path defaults
	if cfg.Paths.BaseImagesDir == "" {
		cfg.Paths.BaseImagesDir = "/var/lib/agentvm/images"
	}
	if cfg.Paths.OverlayDisksDir == "" {
		cfg.Paths.OverlayDisksDir = "/var/lib/agentvm/overlays"
	}
	if cfg.Paths.CloudInitDir == "" {
		cfg.Paths.CloudInitDir = "/var/lib/agentvm/cloud-init"
	}
	if cfg.Paths.DataDir == "" {
		cfg.Paths.DataDir = "/var/lib/agentvm"
	}
	if cfg.Paths.SQLiteDBPath == "" {
		cfg.Paths.SQLiteDBPath = "/var/lib/agentvm/agentvm.db"
	}

	// API defaults
	if cfg.API.ListenAddr == "" {
		cfg.API.ListenAddr = ":8080"
	}
	if cfg.API.ReadTimeout == 0 {
		cfg.API.ReadTimeout = 15 * time.Second
	}
	if cfg.API.WriteTimeout == 0 {
		cfg.API.WriteTimeout = 30 * time.Second
	}
	if cfg.API.ShutdownTimeout == 0 {
		cfg.API.ShutdownTimeout = 30 * time.Second
	}

	// Limits defaults
	if cfg.Limits.MaxConcurrentVMs == 0 {
		cfg.Limits.MaxConcurrentVMs = 10
	}
	if cfg.Limits.MaxConcurrentSessions == 0 {
		cfg.Limits.MaxConcurrentSessions = 50
	}
	if cfg.Limits.VMStartTimeout == 0 {
		cfg.Limits.VMStartTimeout = 60 * time.Second
	}
	if cfg.Limits.DiskSizeDefaultGB == 0 {
		cfg.Limits.DiskSizeDefaultGB = 20
	}

	// Log level default
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	// NOTE: Security.AdminToken has NO default — fail-closed.
}

// ---------------------------------------------------------------------------
// loadFromFile reads a JSON config file and merges into cfg.
func loadFromFile(configPath string, cfg *Config) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	// Use a raw map to handle partial configs gracefully.
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if host, ok := raw["host"].(map[string]any); ok {
		applyMapStringAny(host, map[string]*string{
			"libvirtURI":           &cfg.Host.LibvirtURI,
			"libvirtSocketPath":    &cfg.Host.LibvirtSocketPath,
			"qemuImgPath":          &cfg.Host.QemuImgPath,
			"cloudLocalGenISOPath": &cfg.Host.CloudLocalGenISOPath,
		})
	}

	if paths, ok := raw["paths"].(map[string]any); ok {
		applyMapStringAny(paths, map[string]*string{
			"baseImagesDir":   &cfg.Paths.BaseImagesDir,
			"overlayDisksDir": &cfg.Paths.OverlayDisksDir,
			"cloudInitDir":    &cfg.Paths.CloudInitDir,
			"dataDir":         &cfg.Paths.DataDir,
			"sqliteDBPath":    &cfg.Paths.SQLiteDBPath,
		})
	}

	if api, ok := raw["api"].(map[string]any); ok {
		applyMapStringAny(api, map[string]*string{
			"listenAddr": &cfg.API.ListenAddr,
		})
		if v, ok := api["readTimeout"].(string); ok {
			d, err := time.ParseDuration(v)
			if err != nil {
				return fmt.Errorf("api.readTimeout: invalid duration %q: %w", v, err)
			}
			cfg.API.ReadTimeout = d
		}
		if v, ok := api["writeTimeout"].(string); ok {
			d, err := time.ParseDuration(v)
			if err != nil {
				return fmt.Errorf("api.writeTimeout: invalid duration %q: %w", v, err)
			}
			cfg.API.WriteTimeout = d
		}
		if v, ok := api["shutdownTimeout"].(string); ok {
			d, err := time.ParseDuration(v)
			if err != nil {
				return fmt.Errorf("api.shutdownTimeout: invalid duration %q: %w", v, err)
			}
			cfg.API.ShutdownTimeout = d
		}
	}

	if security, ok := raw["security"].(map[string]any); ok {
		applyMapStringAny(security, map[string]*string{
			"adminToken":  &cfg.Security.AdminToken,
			"tlsCertPath": &cfg.Security.TLSCertPath,
			"tlsKeyPath":  &cfg.Security.TLSKeyPath,
		})
		if v, ok := security["tlsEnabled"].(bool); ok {
			cfg.Security.TLSEnabled = v
		}
	}

	if limits, ok := raw["limits"].(map[string]any); ok {
		if v, ok := limits["maxConcurrentVMs"].(float64); ok {
			cfg.Limits.MaxConcurrentVMs = int(v)
		}
		if v, ok := limits["maxConcurrentSessions"].(float64); ok {
			cfg.Limits.MaxConcurrentSessions = int(v)
		}
		if v, ok := limits["vmStartTimeout"].(string); ok {
			d, err := time.ParseDuration(v)
			if err != nil {
				return fmt.Errorf("limits.vmStartTimeout: invalid duration %q: %w", v, err)
			}
			cfg.Limits.VMStartTimeout = d
		}
		if v, ok := limits["diskSizeDefaultGB"].(float64); ok {
			cfg.Limits.DiskSizeDefaultGB = int(v)
		}
	}

	if v, ok := raw["logLevel"].(string); ok {
		cfg.LogLevel = v
	}

	return nil
}

// applyMapStringAny copies string values from a map to target pointers.
func applyMapStringAny(src map[string]any, targets map[string]*string) {
	for key, ptr := range targets {
		if v, ok := src[key].(string); ok {
			*ptr = v
		}
	}
}

// ---------------------------------------------------------------------------
// loadFromEnv overrides config values from AGENTVM_ environment variables.
func loadFromEnv(cfg *Config) error {
	if v := os.Getenv("AGENTVM_HOST_LIBVIRT_URI"); v != "" {
		cfg.Host.LibvirtURI = v
	}
	if v := os.Getenv("AGENTVM_HOST_LIBVIRT_SOCKET_PATH"); v != "" {
		cfg.Host.LibvirtSocketPath = v
	}
	if v := os.Getenv("AGENTVM_HOST_QEMU_IMG_PATH"); v != "" {
		cfg.Host.QemuImgPath = v
	}
	if v := os.Getenv("AGENTVM_HOST_CLOUD_LOCAL_GEN_ISO_PATH"); v != "" {
		cfg.Host.CloudLocalGenISOPath = v
	}

	if v := os.Getenv("AGENTVM_PATHS_BASE_IMAGES_DIR"); v != "" {
		cfg.Paths.BaseImagesDir = v
	}
	if v := os.Getenv("AGENTVM_PATHS_OVERLAY_DISKS_DIR"); v != "" {
		cfg.Paths.OverlayDisksDir = v
	}
	if v := os.Getenv("AGENTVM_PATHS_CLOUD_INIT_DIR"); v != "" {
		cfg.Paths.CloudInitDir = v
	}
	if v := os.Getenv("AGENTVM_PATHS_DATA_DIR"); v != "" {
		cfg.Paths.DataDir = v
	}
	if v := os.Getenv("AGENTVM_PATHS_SQLITE_DB_PATH"); v != "" {
		cfg.Paths.SQLiteDBPath = v
	}

	if v := os.Getenv("AGENTVM_API_LISTEN_ADDR"); v != "" {
		cfg.API.ListenAddr = v
	}
	if v := os.Getenv("AGENTVM_API_READ_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("AGENTVM_API_READ_TIMEOUT: invalid duration %q: %w", v, err)
		}
		cfg.API.ReadTimeout = d
	}
	if v := os.Getenv("AGENTVM_API_WRITE_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("AGENTVM_API_WRITE_TIMEOUT: invalid duration %q: %w", v, err)
		}
		cfg.API.WriteTimeout = d
	}
	if v := os.Getenv("AGENTVM_API_SHUTDOWN_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("AGENTVM_API_SHUTDOWN_TIMEOUT: invalid duration %q: %w", v, err)
		}
		cfg.API.ShutdownTimeout = d
	}

	if v := os.Getenv("AGENTVM_SECURITY_ADMIN_TOKEN"); v != "" {
		cfg.Security.AdminToken = v
	}
	if v := os.Getenv("AGENTVM_SECURITY_TLS_CERT_PATH"); v != "" {
		cfg.Security.TLSCertPath = v
	}
	if v := os.Getenv("AGENTVM_SECURITY_TLS_KEY_PATH"); v != "" {
		cfg.Security.TLSKeyPath = v
	}
	if v := os.Getenv("AGENTVM_SECURITY_TLS_ENABLED"); v != "" {
		cfg.Security.TLSEnabled = strings.ToLower(v) == "true"
	}

	if v := os.Getenv("AGENTVM_LIMITS_MAX_CONCURRENT_VMS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("AGENTVM_LIMITS_MAX_CONCURRENT_VMS: invalid integer %q: %w", v, err)
		}
		cfg.Limits.MaxConcurrentVMs = n
	}
	if v := os.Getenv("AGENTVM_LIMITS_MAX_CONCURRENT_SESSIONS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("AGENTVM_LIMITS_MAX_CONCURRENT_SESSIONS: invalid integer %q: %w", v, err)
		}
		cfg.Limits.MaxConcurrentSessions = n
	}
	if v := os.Getenv("AGENTVM_LIMITS_VM_START_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("AGENTVM_LIMITS_VM_START_TIMEOUT: invalid duration %q: %w", v, err)
		}
		cfg.Limits.VMStartTimeout = d
	}
	if v := os.Getenv("AGENTVM_LIMITS_DISK_SIZE_DEFAULT_GB"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("AGENTVM_LIMITS_DISK_SIZE_DEFAULT_GB: invalid integer %q: %w", v, err)
		}
		cfg.Limits.DiskSizeDefaultGB = n
	}

	if v := os.Getenv("AGENTVM_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	if v := os.Getenv("AGENTVM_SKIP_HOST_CHECKS"); v != "" {
		cfg.SkipHostChecks = strings.ToLower(v) == "true"
	}

	return nil
}
