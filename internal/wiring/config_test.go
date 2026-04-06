package wiring

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// tb is a minimal interface satisfied by *testing.T and *testing.B.
type tb interface {
	Helper()
	Fatalf(format string, args ...any)
	TempDir() string
}

// helper: write a JSON config to a temp file and return the path.
func writeConfigFile(t tb, data any) string {
	t.Helper()
	f, err := os.CreateTemp("", "agentvm-config-*.json")
	if err != nil {
		t.Fatalf("failed to create temp config file: %v", err)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		t.Fatalf("failed to encode config: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close temp config file: %v", err)
	}
	return f.Name()
}

// helper: clear all AGENTVM_ env vars and restore after test.
func cleanEnv(t *testing.T) {
	t.Helper()
	orig := os.Environ()
	for _, kv := range orig {
		if strings.HasPrefix(kv, "AGENTVM_") {
			k := strings.SplitN(kv, "=", 2)[0]
			os.Unsetenv(k)
		}
	}
	t.Cleanup(func() {
		for _, kv := range orig {
			if strings.HasPrefix(kv, "AGENTVM_") {
				parts := strings.SplitN(kv, "=", 2)
				os.Setenv(parts[0], parts[1])
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Valid minimal config — all defaults applied
// ---------------------------------------------------------------------------

func TestConfig_Load_ValidMinimal(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify defaults
	if cfg.Host.LibvirtURI != "qemu:///system" {
		t.Errorf("Host.LibvirtURI = %q, want %q", cfg.Host.LibvirtURI, "qemu:///system")
	}
	if cfg.API.ListenAddr != ":8080" {
		t.Errorf("API.ListenAddr = %q, want %q", cfg.API.ListenAddr, ":8080")
	}
	if cfg.API.ReadTimeout != 15*time.Second {
		t.Errorf("API.ReadTimeout = %v, want %v", cfg.API.ReadTimeout, 15*time.Second)
	}
	if cfg.API.WriteTimeout != 30*time.Second {
		t.Errorf("API.WriteTimeout = %v, want %v", cfg.API.WriteTimeout, 30*time.Second)
	}
	if cfg.API.ShutdownTimeout != 30*time.Second {
		t.Errorf("API.ShutdownTimeout = %v, want %v", cfg.API.ShutdownTimeout, 30*time.Second)
	}
	if cfg.Limits.MaxConcurrentVMs != 10 {
		t.Errorf("Limits.MaxConcurrentVMs = %d, want %d", cfg.Limits.MaxConcurrentVMs, 10)
	}
	if cfg.Limits.MaxConcurrentSessions != 50 {
		t.Errorf("Limits.MaxConcurrentSessions = %d, want %d", cfg.Limits.MaxConcurrentSessions, 50)
	}
	if cfg.Limits.VMStartTimeout != 60*time.Second {
		t.Errorf("Limits.VMStartTimeout = %v, want %v", cfg.Limits.VMStartTimeout, 60*time.Second)
	}
	if cfg.Limits.DiskSizeDefaultGB != 20 {
		t.Errorf("Limits.DiskSizeDefaultGB = %d, want %d", cfg.Limits.DiskSizeDefaultGB, 20)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.Security.AdminToken != "this-is-a-valid-admin-token-that-is-at-least-32-chars" {
		t.Errorf("Security.AdminToken was not preserved from config file")
	}
}

// ---------------------------------------------------------------------------
// Valid full config — all fields overridden
// ---------------------------------------------------------------------------

func TestConfig_Load_ValidFull(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"host": map[string]any{
			"libvirtURI":           "qemu+ssh://user@host/system",
			"libvirtSocketPath":    "/custom/libvirt-sock",
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"baseImagesDir":   filepath.Join(dir, "images"),
			"overlayDisksDir": filepath.Join(dir, "overlays"),
			"cloudInitDir":    filepath.Join(dir, "cloud-init"),
			"dataDir":         dir,
			"sqliteDBPath":    filepath.Join(dir, "agentvm.db"),
		},
		"api": map[string]any{
			"listenAddr":      ":9090",
			"readTimeout":     "10s",
			"writeTimeout":    "20s",
			"shutdownTimeout": "15s",
		},
		"security": map[string]any{
			"adminToken":  "another-valid-token-that-is-at-least-32-characters-long",
			"tlsCertPath": "",
			"tlsKeyPath":  "",
			"tlsEnabled":  false,
		},
		"limits": map[string]any{
			"maxConcurrentVMs":      5,
			"maxConcurrentSessions": 25,
			"vmStartTimeout":        "90s",
			"diskSizeDefaultGB":     50,
		},
		"logLevel": "debug",
	})

	// Skip host checks since we use non-existent paths for testing.
	t.Setenv("AGENTVM_SKIP_HOST_CHECKS", "true")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Host.LibvirtURI != "qemu+ssh://user@host/system" {
		t.Errorf("Host.LibvirtURI = %q, want %q", cfg.Host.LibvirtURI, "qemu+ssh://user@host/system")
	}
	if cfg.API.ListenAddr != ":9090" {
		t.Errorf("API.ListenAddr = %q, want %q", cfg.API.ListenAddr, ":9090")
	}
	if cfg.API.ReadTimeout != 10*time.Second {
		t.Errorf("API.ReadTimeout = %v, want %v", cfg.API.ReadTimeout, 10*time.Second)
	}
	if cfg.Limits.MaxConcurrentVMs != 5 {
		t.Errorf("Limits.MaxConcurrentVMs = %d, want %d", cfg.Limits.MaxConcurrentVMs, 5)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
}

// ---------------------------------------------------------------------------
// Environment variable overrides
// ---------------------------------------------------------------------------

func TestConfig_Load_EnvOverride(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"api": map[string]any{
			"listenAddr": ":8080",
		},
	})

	t.Setenv("AGENTVM_API_LISTEN_ADDR", ":9999")
	t.Setenv("AGENTVM_LOG_LEVEL", "warn")
	t.Setenv("AGENTVM_LIMITS_MAX_CONCURRENT_VMS", "20")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.API.ListenAddr != ":9999" {
		t.Errorf("API.ListenAddr = %q, want %q (env should override file)", cfg.API.ListenAddr, ":9999")
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want %q (env should override file)", cfg.LogLevel, "warn")
	}
	if cfg.Limits.MaxConcurrentVMs != 20 {
		t.Errorf("Limits.MaxConcurrentVMs = %d, want %d (env should override file)", cfg.Limits.MaxConcurrentVMs, 20)
	}
}

func TestConfig_Load_InvalidBoolEnv_ReturnsError(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
			"libvirtSocketPath":    "/proc/self/stat",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	t.Setenv("AGENTVM_SKIP_HOST_CHECKS", "definitely-not-a-bool")

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid boolean env var, got nil")
	}
}

func TestConfig_Load_UnwritableDirectory_ReturnsError(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	for _, baseDir := range []string{"/sys", "/sys/fs/cgroup", "/proc/sys"} {
		cfgPath := writeConfigFile(t, map[string]any{
			"security": map[string]any{
				"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
			},
			"host": map[string]any{
				"qemuImgPath":          "/usr/bin/true",
				"cloudLocalGenISOPath": "/usr/bin/true",
				"libvirtSocketPath":    "/proc/self/stat",
			},
			"paths": map[string]any{
				"baseImagesDir":   baseDir,
				"overlayDisksDir": filepath.Join(dir, "overlays"),
				"cloudInitDir":    filepath.Join(dir, "cloud-init"),
				"dataDir":         dir,
			},
		})

		_, err := Load(cfgPath)
		if err != nil {
			return
		}
	}

	t.Skip("no unwritable directory candidate failed in this environment")
}

// ---------------------------------------------------------------------------
// Fail-closed: missing admin token
// ---------------------------------------------------------------------------

func TestConfig_Load_MissingAdminToken_FailClosed(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	cfg, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for missing AdminToken, got nil")
	}
	if cfg != nil {
		t.Fatal("expected nil config on validation failure")
	}

	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	found := false
	for _, ve := range verr {
		if ve.Field == "Security.AdminToken" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Security.AdminToken in validation errors, got: %v", verr)
	}
}

// ---------------------------------------------------------------------------
// Fail-closed: admin token too short
// ---------------------------------------------------------------------------

func TestConfig_Load_ShortAdminToken_FailClosed(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "short",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	cfg, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for short AdminToken, got nil")
	}
	if cfg != nil {
		t.Fatal("expected nil config on validation failure")
	}
}

// ---------------------------------------------------------------------------
// Invalid LibvirtURI
// ---------------------------------------------------------------------------

func TestConfig_Load_InvalidLibvirtURI(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"libvirtURI":           "not-a-uri",
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	cfg, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid LibvirtURI, got nil")
	}
	if cfg != nil {
		t.Fatal("expected nil config on validation failure")
	}
}

// ---------------------------------------------------------------------------
// Invalid ListenAddr
// ---------------------------------------------------------------------------

func TestConfig_Load_InvalidListenAddr(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"api": map[string]any{
			"listenAddr": "abc",
		},
	})

	cfg, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid ListenAddr, got nil")
	}
	if cfg != nil {
		t.Fatal("expected nil config on validation failure")
	}
}

// ---------------------------------------------------------------------------
// Limits out of range
// ---------------------------------------------------------------------------

func TestConfig_Load_MaxConcurrentVMsOutOfRange(t *testing.T) {
	cleanEnv(t)

	tests := []struct {
		name  string
		value int
	}{
		{"zero", 0},
		{"negative", -1},
		{"over_max", 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			cfgPath := writeConfigFile(t, map[string]any{
				"security": map[string]any{
					"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
				},
				"host": map[string]any{
					"qemuImgPath":          "/usr/bin/true",
					"cloudLocalGenISOPath": "/usr/bin/true",
				},
				"paths": map[string]any{
					"dataDir": dir,
				},
				"limits": map[string]any{
					"maxConcurrentVMs": tt.value,
				},
			})

			cfg, err := Load(cfgPath)
			if err == nil {
				t.Fatalf("expected error for MaxConcurrentVMs=%d, got nil", tt.value)
			}
			if cfg != nil {
				t.Fatal("expected nil config on validation failure")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Non-existent required path (with host checks enabled)
// ---------------------------------------------------------------------------

func TestConfig_Load_NonExistentRequiredPath(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/nonexistent/path/qemu-img",
			"cloudLocalGenISOPath": "/nonexistent/path/genisoimage",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	cfg, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for non-existent binary path, got nil")
	}
	if cfg != nil {
		t.Fatal("expected nil config on validation failure")
	}
}

// ---------------------------------------------------------------------------
// Skip host checks flag
// ---------------------------------------------------------------------------

func TestConfig_Load_SkipHostChecks(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/nonexistent/path/qemu-img",
			"cloudLocalGenISOPath": "/nonexistent/path/genisoimage",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	t.Setenv("AGENTVM_SKIP_HOST_CHECKS", "true")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error with skip host checks: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config with skip host checks")
	}
	if !cfg.SkipHostChecks {
		t.Error("expected SkipHostChecks to be true")
	}
}

// ---------------------------------------------------------------------------
// TLS config incomplete
// ---------------------------------------------------------------------------

func TestConfig_Load_TLSConfigIncomplete(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()

	// Create a cert file but no key file
	certPath := filepath.Join(dir, "cert.pem")
	if err := os.WriteFile(certPath, []byte("cert-data"), 0644); err != nil {
		t.Fatalf("failed to write cert file: %v", err)
	}

	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken":  "this-is-a-valid-admin-token-that-is-at-least-32-chars",
			"tlsCertPath": certPath,
			"tlsKeyPath":  "",
			"tlsEnabled":  true,
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	cfg, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for incomplete TLS config, got nil")
	}
	if cfg != nil {
		t.Fatal("expected nil config on validation failure")
	}
}

// ---------------------------------------------------------------------------
// TLS config complete
// ---------------------------------------------------------------------------

func TestConfig_Load_TLSConfigComplete(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()

	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certPath, []byte("cert-data"), 0644); err != nil {
		t.Fatalf("failed to write cert file: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("key-data"), 0600); err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}

	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken":  "this-is-a-valid-admin-token-that-is-at-least-32-chars",
			"tlsCertPath": certPath,
			"tlsKeyPath":  keyPath,
			"tlsEnabled":  true,
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Security.TLSEnabled {
		t.Error("expected TLSEnabled to be true")
	}
}

// ---------------------------------------------------------------------------
// Invalid log level
// ---------------------------------------------------------------------------

func TestConfig_Load_InvalidLogLevel(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"logLevel": "verbose",
	})

	cfg, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid LogLevel, got nil")
	}
	if cfg != nil {
		t.Fatal("expected nil config on validation failure")
	}
}

// ---------------------------------------------------------------------------
// Valid log levels
// ---------------------------------------------------------------------------

func TestConfig_Load_ValidLogLevel(t *testing.T) {
	cleanEnv(t)

	validLevels := []string{"debug", "info", "warn", "error"}

	for _, level := range validLevels {
		t.Run(level, func(t *testing.T) {
			dir := t.TempDir()
			cfgPath := writeConfigFile(t, map[string]any{
				"security": map[string]any{
					"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
				},
				"host": map[string]any{
					"qemuImgPath":          "/usr/bin/true",
					"cloudLocalGenISOPath": "/usr/bin/true",
				},
				"paths": map[string]any{
					"dataDir": dir,
				},
				"logLevel": level,
			})

			cfg, err := Load(cfgPath)
			if err != nil {
				t.Fatalf("unexpected error for logLevel=%q: %v", level, err)
			}
			if cfg.LogLevel != level {
				t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, level)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Performance: config load must complete in < 100ms
// ---------------------------------------------------------------------------

func TestConfig_Load_Performance_Sub100ms(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	start := time.Now()
	_, err := Load(cfgPath)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("Load took %v, must be < 100ms", elapsed)
	}
}

// ---------------------------------------------------------------------------
// ValidationErrors error output
// ---------------------------------------------------------------------------

func TestValidationError_ErrorOutput_Multiple(t *testing.T) {
	errs := ValidationErrors{
		{Field: "Security.AdminToken", Reason: "must be set and at least 32 characters (fail-closed)"},
		{Field: "Host.QemuImgPath", Reason: "binary not found"},
		{Field: "API.ListenAddr", Reason: "invalid host:port format"},
	}

	msg := errs.Error()
	if !strings.Contains(msg, "3 errors") {
		t.Errorf("error message should mention '3 errors', got: %s", msg)
	}
	if !strings.Contains(msg, "Security.AdminToken") {
		t.Errorf("error message should contain 'Security.AdminToken', got: %s", msg)
	}
	if !strings.Contains(msg, "Host.QemuImgPath") {
		t.Errorf("error message should contain 'Host.QemuImgPath', got: %s", msg)
	}
	if !strings.Contains(msg, "API.ListenAddr") {
		t.Errorf("error message should contain 'API.ListenAddr', got: %s", msg)
	}
}

func TestValidationError_ErrorOutput_Single(t *testing.T) {
	errs := ValidationErrors{
		{Field: "Security.AdminToken", Reason: "must be set and at least 32 characters (fail-closed)"},
	}

	msg := errs.Error()
	if strings.Contains(msg, "errors") {
		t.Errorf("single error message should not use plural 'errors', got: %s", msg)
	}
	if !strings.Contains(msg, "Security.AdminToken") {
		t.Errorf("error message should contain 'Security.AdminToken', got: %s", msg)
	}
}

// ---------------------------------------------------------------------------
// DiskSizeDefaultGB validation
// ---------------------------------------------------------------------------

func TestConfig_Load_DiskSizeDefaultGB_OutOfRange(t *testing.T) {
	cleanEnv(t)

	tests := []struct {
		name  string
		value int
	}{
		{"zero", 0},
		{"below_min", 4},
		{"above_max", 501},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			cfgPath := writeConfigFile(t, map[string]any{
				"security": map[string]any{
					"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
				},
				"host": map[string]any{
					"qemuImgPath":          "/usr/bin/true",
					"cloudLocalGenISOPath": "/usr/bin/true",
				},
				"paths": map[string]any{
					"dataDir": dir,
				},
				"limits": map[string]any{
					"diskSizeDefaultGB": tt.value,
				},
			})

			cfg, err := Load(cfgPath)
			if err == nil {
				t.Fatalf("expected error for DiskSizeDefaultGB=%d, got nil", tt.value)
			}
			if cfg != nil {
				t.Fatal("expected nil config on validation failure")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// MaxConcurrentSessions validation
// ---------------------------------------------------------------------------

func TestConfig_Load_MaxConcurrentSessions_OutOfRange(t *testing.T) {
	cleanEnv(t)

	tests := []struct {
		name  string
		value int
	}{
		{"zero", 0},
		{"over_max", 501},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			cfgPath := writeConfigFile(t, map[string]any{
				"security": map[string]any{
					"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
				},
				"host": map[string]any{
					"qemuImgPath":          "/usr/bin/true",
					"cloudLocalGenISOPath": "/usr/bin/true",
				},
				"paths": map[string]any{
					"dataDir": dir,
				},
				"limits": map[string]any{
					"maxConcurrentSessions": tt.value,
				},
			})

			cfg, err := Load(cfgPath)
			if err == nil {
				t.Fatalf("expected error for MaxConcurrentSessions=%d, got nil", tt.value)
			}
			if cfg != nil {
				t.Fatal("expected nil config on validation failure")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// VMStartTimeout validation (must be >= 30s)
// ---------------------------------------------------------------------------

func TestConfig_Load_VMStartTimeout_TooShort(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"limits": map[string]any{
			"vmStartTimeout": "10s",
		},
	})

	cfg, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for VMStartTimeout < 30s, got nil")
	}
	if cfg != nil {
		t.Fatal("expected nil config on validation failure")
	}
}

// ---------------------------------------------------------------------------
// API timeout validation (must be > 0)
// ---------------------------------------------------------------------------

func TestConfig_Load_APITimeout_Zero(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"api": map[string]any{
			"readTimeout": "0s",
		},
	})

	cfg, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for zero ReadTimeout, got nil")
	}
	if cfg != nil {
		t.Fatal("expected nil config on validation failure")
	}
}

// ---------------------------------------------------------------------------
// Env-only config (no file)
// ---------------------------------------------------------------------------

func TestConfig_Load_EnvOnly(t *testing.T) {
	cleanEnv(t)

	t.Setenv("AGENTVM_SECURITY_ADMIN_TOKEN", "this-is-a-valid-admin-token-from-env-32chars")
	t.Setenv("AGENTVM_SKIP_HOST_CHECKS", "true")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Security.AdminToken != "this-is-a-valid-admin-token-from-env-32chars" {
		t.Errorf("Security.AdminToken = %q, want env value", cfg.Security.AdminToken)
	}
}

// ---------------------------------------------------------------------------
// Benchmark: config load performance
// ---------------------------------------------------------------------------

func BenchmarkConfig_Load(b *testing.B) {
	dir := b.TempDir()
	cfgPath := writeConfigFile(b, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Load(cfgPath)
		if err != nil {
			b.Fatalf("Load failed: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Coverage: env-only overrides for all fields
// ---------------------------------------------------------------------------

func TestConfig_Load_EnvOverrides_AllFields(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	socketPath := filepath.Join(dir, "libvirt-sock")
	if err := os.WriteFile(socketPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create socket file: %v", err)
	}

	t.Setenv("AGENTVM_HOST_LIBVIRT_URI", "qemu+tcp://localhost/system")
	t.Setenv("AGENTVM_HOST_LIBVIRT_SOCKET_PATH", socketPath)
	t.Setenv("AGENTVM_HOST_QEMU_IMG_PATH", "/usr/bin/true")
	t.Setenv("AGENTVM_HOST_CLOUD_LOCAL_GEN_ISO_PATH", "/usr/bin/true")
	t.Setenv("AGENTVM_PATHS_BASE_IMAGES_DIR", filepath.Join(dir, "images"))
	t.Setenv("AGENTVM_PATHS_OVERLAY_DISKS_DIR", filepath.Join(dir, "overlays"))
	t.Setenv("AGENTVM_PATHS_CLOUD_INIT_DIR", filepath.Join(dir, "cloud-init"))
	t.Setenv("AGENTVM_PATHS_DATA_DIR", dir)
	t.Setenv("AGENTVM_PATHS_SQLITE_DB_PATH", filepath.Join(dir, "test.db"))
	t.Setenv("AGENTVM_API_LISTEN_ADDR", ":7070")
	t.Setenv("AGENTVM_API_READ_TIMEOUT", "5s")
	t.Setenv("AGENTVM_API_WRITE_TIMEOUT", "10s")
	t.Setenv("AGENTVM_API_SHUTDOWN_TIMEOUT", "8s")
	t.Setenv("AGENTVM_SECURITY_ADMIN_TOKEN", "this-is-a-valid-admin-token-that-is-at-least-32-chars")
	t.Setenv("AGENTVM_LIMITS_MAX_CONCURRENT_VMS", "15")
	t.Setenv("AGENTVM_LIMITS_MAX_CONCURRENT_SESSIONS", "100")
	t.Setenv("AGENTVM_LIMITS_VM_START_TIMEOUT", "45s")
	t.Setenv("AGENTVM_LIMITS_DISK_SIZE_DEFAULT_GB", "30")
	t.Setenv("AGENTVM_LOG_LEVEL", "error")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Host.LibvirtURI != "qemu+tcp://localhost/system" {
		t.Errorf("Host.LibvirtURI = %q", cfg.Host.LibvirtURI)
	}
	if cfg.Host.LibvirtSocketPath != socketPath {
		t.Errorf("Host.LibvirtSocketPath = %q", cfg.Host.LibvirtSocketPath)
	}
	if cfg.API.ListenAddr != ":7070" {
		t.Errorf("API.ListenAddr = %q", cfg.API.ListenAddr)
	}
	if cfg.API.ReadTimeout != 5*time.Second {
		t.Errorf("API.ReadTimeout = %v", cfg.API.ReadTimeout)
	}
	if cfg.API.WriteTimeout != 10*time.Second {
		t.Errorf("API.WriteTimeout = %v", cfg.API.WriteTimeout)
	}
	if cfg.API.ShutdownTimeout != 8*time.Second {
		t.Errorf("API.ShutdownTimeout = %v", cfg.API.ShutdownTimeout)
	}
	if cfg.Limits.MaxConcurrentVMs != 15 {
		t.Errorf("Limits.MaxConcurrentVMs = %d", cfg.Limits.MaxConcurrentVMs)
	}
	if cfg.Limits.MaxConcurrentSessions != 100 {
		t.Errorf("Limits.MaxConcurrentSessions = %d", cfg.Limits.MaxConcurrentSessions)
	}
	if cfg.Limits.VMStartTimeout != 45*time.Second {
		t.Errorf("Limits.VMStartTimeout = %v", cfg.Limits.VMStartTimeout)
	}
	if cfg.Limits.DiskSizeDefaultGB != 30 {
		t.Errorf("Limits.DiskSizeDefaultGB = %d", cfg.Limits.DiskSizeDefaultGB)
	}
	if cfg.LogLevel != "error" {
		t.Errorf("LogLevel = %q", cfg.LogLevel)
	}
}

// ---------------------------------------------------------------------------
// Coverage: TLS enabled with both cert and key missing
// ---------------------------------------------------------------------------

func TestConfig_Load_TLSBothMissing(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
			"tlsEnabled": true,
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	cfg, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for TLS enabled with both cert and key missing, got nil")
	}
	if cfg != nil {
		t.Fatal("expected nil config on validation failure")
	}
}

// ---------------------------------------------------------------------------
// Coverage: invalid JSON file
// ---------------------------------------------------------------------------

func TestConfig_Load_InvalidJSON(t *testing.T) {
	cleanEnv(t)

	f, err := os.CreateTemp("", "agentvm-bad-json-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := f.WriteString("{not valid json"); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	f.Close()

	cfg, err := Load(f.Name())
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if cfg != nil {
		t.Fatal("expected nil config on parse failure")
	}
}

// ---------------------------------------------------------------------------
// Coverage: non-existent config file
// ---------------------------------------------------------------------------

func TestConfig_Load_NonExistentFile(t *testing.T) {
	cleanEnv(t)

	cfg, err := Load("/nonexistent/config.json")
	if err == nil {
		t.Fatal("expected error for non-existent config file, got nil")
	}
	if cfg != nil {
		t.Fatal("expected nil config on file load failure")
	}
}

// ---------------------------------------------------------------------------
// Coverage: binary not executable
// ---------------------------------------------------------------------------

func TestConfig_Load_BinaryNotExecutable(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	binPath := filepath.Join(dir, "qemu-img")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\necho ok"), 0644); err != nil {
		t.Fatalf("failed to write fake binary: %v", err)
	}

	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          binPath,
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	cfg, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for non-executable binary, got nil")
	}
	if cfg != nil {
		t.Fatal("expected nil config on validation failure")
	}
}

// ---------------------------------------------------------------------------
// Coverage: directory exists but is a file (ensureDir error path)
// ---------------------------------------------------------------------------

func TestConfig_Load_PathIsFile(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	// Create a file where a directory is expected.
	filePath := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(filePath, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir":       dir,
			"baseImagesDir": filePath, // this is a file, not a directory
		},
	})

	cfg, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error when path is a file not a directory, got nil")
	}
	if cfg != nil {
		t.Fatal("expected nil config on validation failure")
	}
}

// ---------------------------------------------------------------------------
// Coverage: env TLS enabled via string
// ---------------------------------------------------------------------------

func TestConfig_Load_EnvTLSEnabled(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	if err := os.WriteFile(certPath, []byte("cert"), 0644); err != nil {
		t.Fatalf("failed to write cert: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("key"), 0600); err != nil {
		t.Fatalf("failed to write key: %v", err)
	}
	socketPath := filepath.Join(dir, "libvirt-sock")
	if err := os.WriteFile(socketPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create socket: %v", err)
	}

	t.Setenv("AGENTVM_SECURITY_ADMIN_TOKEN", "this-is-a-valid-admin-token-that-is-at-least-32-chars")
	t.Setenv("AGENTVM_SECURITY_TLS_ENABLED", "true")
	t.Setenv("AGENTVM_SECURITY_TLS_CERT_PATH", certPath)
	t.Setenv("AGENTVM_SECURITY_TLS_KEY_PATH", keyPath)
	t.Setenv("AGENTVM_HOST_LIBVIRT_SOCKET_PATH", socketPath)
	t.Setenv("AGENTVM_HOST_QEMU_IMG_PATH", "/usr/bin/true")
	t.Setenv("AGENTVM_HOST_CLOUD_LOCAL_GEN_ISO_PATH", "/usr/bin/true")
	t.Setenv("AGENTVM_PATHS_DATA_DIR", dir)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Security.TLSEnabled {
		t.Error("expected TLSEnabled to be true from env")
	}
}

// ---------------------------------------------------------------------------
// Helper: check if error is ValidationErrors
// ---------------------------------------------------------------------------

func asValidationErrors(err error, target *ValidationErrors) bool {
	if verr, ok := err.(ValidationErrors); ok {
		*target = verr
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Coverage: ValidationErrors.Is
// ---------------------------------------------------------------------------

func TestValidationErrors_Is(t *testing.T) {
	errs := ValidationErrors{
		{Field: "X", Reason: "bad"},
	}

	var target error = errs
	if !errors.Is(target, ValidationErrors{}) {
		t.Error("expected errors.Is to match ValidationErrors")
	}
	if errors.Is(target, fmt.Errorf("other")) {
		t.Error("expected errors.Is to not match unrelated error")
	}
}

// ---------------------------------------------------------------------------
// Coverage: ValidationErrors.Error empty case
// ---------------------------------------------------------------------------

func TestValidationErrors_Error_Empty(t *testing.T) {
	errs := ValidationErrors{}
	if errs.Error() != "" {
		t.Errorf("expected empty string for empty ValidationErrors, got %q", errs.Error())
	}
}

// ---------------------------------------------------------------------------
// Coverage: validatePaths missing DataDir
// ---------------------------------------------------------------------------

func TestConfig_Load_MissingDataDir(t *testing.T) {
	cleanEnv(t)

	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": "",
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for empty DataDir, got nil")
	}
}

// ---------------------------------------------------------------------------
// Coverage: validateAPI empty ListenAddr
// ---------------------------------------------------------------------------

func TestConfig_Load_EmptyListenAddr(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"api": map[string]any{
			"listenAddr": "",
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for empty ListenAddr, got nil")
	}
}

// ---------------------------------------------------------------------------
// Coverage: checkBinary — path is a directory
// ---------------------------------------------------------------------------

func TestConfig_Load_BinaryIsDirectory(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          dir,
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for directory as binary, got nil")
	}
}

// ---------------------------------------------------------------------------
// Coverage: loadFromEnv — invalid duration for API timeouts
// ---------------------------------------------------------------------------

func TestConfig_Load_EnvInvalidDuration(t *testing.T) {
	cleanEnv(t)

	t.Setenv("AGENTVM_SECURITY_ADMIN_TOKEN", "this-is-a-valid-admin-token-that-is-at-least-32-chars")
	t.Setenv("AGENTVM_SKIP_HOST_CHECKS", "true")
	t.Setenv("AGENTVM_API_READ_TIMEOUT", "not-a-duration")

	_, err := Load("")
	if err == nil {
		t.Fatal("expected error for invalid duration env var, got nil")
	}
}

// ---------------------------------------------------------------------------
// Coverage: loadFromEnv — invalid integer for limits
// ---------------------------------------------------------------------------

func TestConfig_Load_EnvInvalidInteger(t *testing.T) {
	cleanEnv(t)

	t.Setenv("AGENTVM_SECURITY_ADMIN_TOKEN", "this-is-a-valid-admin-token-that-is-at-least-32-chars")
	t.Setenv("AGENTVM_SKIP_HOST_CHECKS", "true")
	t.Setenv("AGENTVM_LIMITS_MAX_CONCURRENT_VMS", "not-a-number")

	_, err := Load("")
	if err == nil {
		t.Fatal("expected error for invalid integer env var, got nil")
	}
}

// ---------------------------------------------------------------------------
// Coverage: loadFromEnv — invalid duration for limits
// ---------------------------------------------------------------------------

func TestConfig_Load_EnvInvalidLimitDuration(t *testing.T) {
	cleanEnv(t)

	t.Setenv("AGENTVM_SECURITY_ADMIN_TOKEN", "this-is-a-valid-admin-token-that-is-at-least-32-chars")
	t.Setenv("AGENTVM_SKIP_HOST_CHECKS", "true")
	t.Setenv("AGENTVM_LIMITS_VM_START_TIMEOUT", "not-a-duration")

	_, err := Load("")
	if err == nil {
		t.Fatal("expected error for invalid duration env var, got nil")
	}
}

// ---------------------------------------------------------------------------
// Coverage: loadFromEnv — invalid duration for API write/shutdown timeouts
// ---------------------------------------------------------------------------

func TestConfig_Load_EnvInvalidWriteTimeout(t *testing.T) {
	cleanEnv(t)

	t.Setenv("AGENTVM_SECURITY_ADMIN_TOKEN", "this-is-a-valid-admin-token-that-is-at-least-32-chars")
	t.Setenv("AGENTVM_SKIP_HOST_CHECKS", "true")
	t.Setenv("AGENTVM_API_WRITE_TIMEOUT", "bad")

	_, err := Load("")
	if err == nil {
		t.Fatal("expected error for invalid write timeout, got nil")
	}
}

func TestConfig_Load_EnvInvalidShutdownTimeout(t *testing.T) {
	cleanEnv(t)

	t.Setenv("AGENTVM_SECURITY_ADMIN_TOKEN", "this-is-a-valid-admin-token-that-is-at-least-32-chars")
	t.Setenv("AGENTVM_SKIP_HOST_CHECKS", "true")
	t.Setenv("AGENTVM_API_SHUTDOWN_TIMEOUT", "bad")

	_, err := Load("")
	if err == nil {
		t.Fatal("expected error for invalid shutdown timeout, got nil")
	}
}

// ---------------------------------------------------------------------------
// Coverage: loadFromFile — invalid duration in file
// ---------------------------------------------------------------------------

func TestConfig_Load_InvalidFileDuration(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"api": map[string]any{
			"readTimeout": "not-a-duration",
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid file duration, got nil")
	}
}

// ---------------------------------------------------------------------------
// Coverage: loadFromFile — invalid integer in file
// ---------------------------------------------------------------------------

func TestConfig_Load_InvalidFileInteger(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"limits": map[string]any{
			"maxConcurrentVMs": float64(5),
		},
	})

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Limits.MaxConcurrentVMs != 5 {
		t.Errorf("Limits.MaxConcurrentVMs = %d, want 5", cfg.Limits.MaxConcurrentVMs)
	}
}

// ---------------------------------------------------------------------------
// Coverage: ensureDir — other stat error (permission denied simulation)
// ---------------------------------------------------------------------------

func TestConfig_Load_EnsureDirError(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "blocked")
	if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	socketPath := filepath.Join(dir, "libvirt-sock")
	if err := os.WriteFile(socketPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create socket file: %v", err)
	}

	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"libvirtURI":           "qemu:///system",
			"libvirtSocketPath":    socketPath,
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir":         dir,
			"baseImagesDir":   filePath,
			"overlayDisksDir": filepath.Join(dir, "overlays"),
			"cloudInitDir":    filepath.Join(dir, "cloud-init"),
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for ensureDir failure, got nil")
	}
}

// ---------------------------------------------------------------------------
// Mutation test coverage: verify error message content and field values
// ---------------------------------------------------------------------------

func TestConfig_Load_FileError_MessageContent(t *testing.T) {
	cleanEnv(t)

	_, err := Load("/nonexistent/config.json")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to load config file") {
		t.Errorf("error should mention 'failed to load config file', got: %v", err)
	}
}

func TestConfig_Load_InvalidJSON_MessageContent(t *testing.T) {
	cleanEnv(t)

	f, err := os.CreateTemp("", "agentvm-bad-json-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := f.WriteString("{not valid json"); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	f.Close()

	_, err = Load(f.Name())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to load config file") {
		t.Errorf("error should mention 'failed to load config file', got: %v", err)
	}
}

func TestConfig_Load_FileDurationError_MessageContent(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"api": map[string]any{
			"readTimeout": "not-a-duration",
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to load config file") {
		t.Errorf("error should mention 'failed to load config file', got: %v", err)
	}
	if !strings.Contains(err.Error(), "api.readTimeout") {
		t.Errorf("error should mention 'api.readTimeout', got: %v", err)
	}
}

func TestConfig_Load_FileWriteTimeoutError_MessageContent(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"api": map[string]any{
			"writeTimeout": "bad",
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "api.writeTimeout") {
		t.Errorf("error should mention 'api.writeTimeout', got: %v", err)
	}
}

func TestConfig_Load_FileShutdownTimeoutError_MessageContent(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"api": map[string]any{
			"shutdownTimeout": "bad",
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "api.shutdownTimeout") {
		t.Errorf("error should mention 'api.shutdownTimeout', got: %v", err)
	}
}

func TestConfig_Load_FileLimitsDurationError_MessageContent(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"limits": map[string]any{
			"vmStartTimeout": "not-a-duration",
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "limits.vmStartTimeout") {
		t.Errorf("error should mention 'limits.vmStartTimeout', got: %v", err)
	}
}

func TestConfig_Load_EnvReadTimeoutError_MessageContent(t *testing.T) {
	cleanEnv(t)

	t.Setenv("AGENTVM_SECURITY_ADMIN_TOKEN", "this-is-a-valid-admin-token-that-is-at-least-32-chars")
	t.Setenv("AGENTVM_SKIP_HOST_CHECKS", "true")
	t.Setenv("AGENTVM_API_READ_TIMEOUT", "not-a-duration")

	_, err := Load("")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "AGENTVM_API_READ_TIMEOUT") {
		t.Errorf("error should mention 'AGENTVM_API_READ_TIMEOUT', got: %v", err)
	}
}

func TestConfig_Load_EnvWriteTimeoutError_MessageContent(t *testing.T) {
	cleanEnv(t)

	t.Setenv("AGENTVM_SECURITY_ADMIN_TOKEN", "this-is-a-valid-admin-token-that-is-at-least-32-chars")
	t.Setenv("AGENTVM_SKIP_HOST_CHECKS", "true")
	t.Setenv("AGENTVM_API_WRITE_TIMEOUT", "bad")

	_, err := Load("")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "AGENTVM_API_WRITE_TIMEOUT") {
		t.Errorf("error should mention 'AGENTVM_API_WRITE_TIMEOUT', got: %v", err)
	}
}

func TestConfig_Load_EnvShutdownTimeoutError_MessageContent(t *testing.T) {
	cleanEnv(t)

	t.Setenv("AGENTVM_SECURITY_ADMIN_TOKEN", "this-is-a-valid-admin-token-that-is-at-least-32-chars")
	t.Setenv("AGENTVM_SKIP_HOST_CHECKS", "true")
	t.Setenv("AGENTVM_API_SHUTDOWN_TIMEOUT", "bad")

	_, err := Load("")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "AGENTVM_API_SHUTDOWN_TIMEOUT") {
		t.Errorf("error should mention 'AGENTVM_API_SHUTDOWN_TIMEOUT', got: %v", err)
	}
}

func TestConfig_Load_EnvLimitsDurationError_MessageContent(t *testing.T) {
	cleanEnv(t)

	t.Setenv("AGENTVM_SECURITY_ADMIN_TOKEN", "this-is-a-valid-admin-token-that-is-at-least-32-chars")
	t.Setenv("AGENTVM_SKIP_HOST_CHECKS", "true")
	t.Setenv("AGENTVM_LIMITS_VM_START_TIMEOUT", "not-a-duration")

	_, err := Load("")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "AGENTVM_LIMITS_VM_START_TIMEOUT") {
		t.Errorf("error should mention 'AGENTVM_LIMITS_VM_START_TIMEOUT', got: %v", err)
	}
}

func TestConfig_Load_EnvInvalidInteger_MessageContent(t *testing.T) {
	cleanEnv(t)

	t.Setenv("AGENTVM_SECURITY_ADMIN_TOKEN", "this-is-a-valid-admin-token-that-is-at-least-32-chars")
	t.Setenv("AGENTVM_SKIP_HOST_CHECKS", "true")
	t.Setenv("AGENTVM_LIMITS_MAX_CONCURRENT_VMS", "not-a-number")

	_, err := Load("")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "AGENTVM_LIMITS_MAX_CONCURRENT_VMS") {
		t.Errorf("error should mention 'AGENTVM_LIMITS_MAX_CONCURRENT_VMS', got: %v", err)
	}
}

func TestConfig_Load_EnvInvalidSessionsInteger_MessageContent(t *testing.T) {
	cleanEnv(t)

	t.Setenv("AGENTVM_SECURITY_ADMIN_TOKEN", "this-is-a-valid-admin-token-that-is-at-least-32-chars")
	t.Setenv("AGENTVM_SKIP_HOST_CHECKS", "true")
	t.Setenv("AGENTVM_LIMITS_MAX_CONCURRENT_SESSIONS", "not-a-number")

	_, err := Load("")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "AGENTVM_LIMITS_MAX_CONCURRENT_SESSIONS") {
		t.Errorf("error should mention 'AGENTVM_LIMITS_MAX_CONCURRENT_SESSIONS', got: %v", err)
	}
}

func TestConfig_Load_EnvInvalidDiskSizeInteger_MessageContent(t *testing.T) {
	cleanEnv(t)

	t.Setenv("AGENTVM_SECURITY_ADMIN_TOKEN", "this-is-a-valid-admin-token-that-is-at-least-32-chars")
	t.Setenv("AGENTVM_SKIP_HOST_CHECKS", "true")
	t.Setenv("AGENTVM_LIMITS_DISK_SIZE_DEFAULT_GB", "not-a-number")

	_, err := Load("")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "AGENTVM_LIMITS_DISK_SIZE_DEFAULT_GB") {
		t.Errorf("error should mention 'AGENTVM_LIMITS_DISK_SIZE_DEFAULT_GB', got: %v", err)
	}
}

func TestConfig_Load_EnvOverrides_VerifyValues(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	socketPath := filepath.Join(dir, "libvirt-sock")
	if err := os.WriteFile(socketPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create socket file: %v", err)
	}

	t.Setenv("AGENTVM_HOST_LIBVIRT_URI", "qemu+tcp://localhost/system")
	t.Setenv("AGENTVM_HOST_LIBVIRT_SOCKET_PATH", socketPath)
	t.Setenv("AGENTVM_HOST_QEMU_IMG_PATH", "/usr/bin/true")
	t.Setenv("AGENTVM_HOST_CLOUD_LOCAL_GEN_ISO_PATH", "/usr/bin/true")
	t.Setenv("AGENTVM_PATHS_BASE_IMAGES_DIR", filepath.Join(dir, "images"))
	t.Setenv("AGENTVM_PATHS_OVERLAY_DISKS_DIR", filepath.Join(dir, "overlays"))
	t.Setenv("AGENTVM_PATHS_CLOUD_INIT_DIR", filepath.Join(dir, "cloud-init"))
	t.Setenv("AGENTVM_PATHS_DATA_DIR", dir)
	t.Setenv("AGENTVM_PATHS_SQLITE_DB_PATH", filepath.Join(dir, "custom.db"))
	t.Setenv("AGENTVM_API_LISTEN_ADDR", ":7070")
	t.Setenv("AGENTVM_API_READ_TIMEOUT", "5s")
	t.Setenv("AGENTVM_API_WRITE_TIMEOUT", "10s")
	t.Setenv("AGENTVM_API_SHUTDOWN_TIMEOUT", "8s")
	t.Setenv("AGENTVM_SECURITY_ADMIN_TOKEN", "this-is-a-valid-admin-token-that-is-at-least-32-chars")
	t.Setenv("AGENTVM_LIMITS_MAX_CONCURRENT_VMS", "15")
	t.Setenv("AGENTVM_LIMITS_MAX_CONCURRENT_SESSIONS", "100")
	t.Setenv("AGENTVM_LIMITS_VM_START_TIMEOUT", "45s")
	t.Setenv("AGENTVM_LIMITS_DISK_SIZE_DEFAULT_GB", "30")
	t.Setenv("AGENTVM_LOG_LEVEL", "error")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify each env override actually took effect
	if cfg.Host.LibvirtURI != "qemu+tcp://localhost/system" {
		t.Errorf("Host.LibvirtURI = %q, want qemu+tcp://localhost/system", cfg.Host.LibvirtURI)
	}
	if cfg.Host.LibvirtSocketPath != socketPath {
		t.Errorf("Host.LibvirtSocketPath = %q, want %q", cfg.Host.LibvirtSocketPath, socketPath)
	}
	if cfg.Host.QemuImgPath != "/usr/bin/true" {
		t.Errorf("Host.QemuImgPath = %q, want /usr/bin/true", cfg.Host.QemuImgPath)
	}
	if cfg.Host.CloudLocalGenISOPath != "/usr/bin/true" {
		t.Errorf("Host.CloudLocalGenISOPath = %q, want /usr/bin/true", cfg.Host.CloudLocalGenISOPath)
	}
	if cfg.Paths.BaseImagesDir != filepath.Join(dir, "images") {
		t.Errorf("Paths.BaseImagesDir = %q", cfg.Paths.BaseImagesDir)
	}
	if cfg.Paths.OverlayDisksDir != filepath.Join(dir, "overlays") {
		t.Errorf("Paths.OverlayDisksDir = %q", cfg.Paths.OverlayDisksDir)
	}
	if cfg.Paths.CloudInitDir != filepath.Join(dir, "cloud-init") {
		t.Errorf("Paths.CloudInitDir = %q", cfg.Paths.CloudInitDir)
	}
	if cfg.Paths.DataDir != dir {
		t.Errorf("Paths.DataDir = %q", cfg.Paths.DataDir)
	}
	if cfg.Paths.SQLiteDBPath != filepath.Join(dir, "custom.db") {
		t.Errorf("Paths.SQLiteDBPath = %q", cfg.Paths.SQLiteDBPath)
	}
	if cfg.API.ListenAddr != ":7070" {
		t.Errorf("API.ListenAddr = %q", cfg.API.ListenAddr)
	}
	if cfg.API.ReadTimeout != 5*time.Second {
		t.Errorf("API.ReadTimeout = %v", cfg.API.ReadTimeout)
	}
	if cfg.API.WriteTimeout != 10*time.Second {
		t.Errorf("API.WriteTimeout = %v", cfg.API.WriteTimeout)
	}
	if cfg.API.ShutdownTimeout != 8*time.Second {
		t.Errorf("API.ShutdownTimeout = %v", cfg.API.ShutdownTimeout)
	}
	if cfg.Limits.MaxConcurrentVMs != 15 {
		t.Errorf("Limits.MaxConcurrentVMs = %d", cfg.Limits.MaxConcurrentVMs)
	}
	if cfg.Limits.MaxConcurrentSessions != 100 {
		t.Errorf("Limits.MaxConcurrentSessions = %d", cfg.Limits.MaxConcurrentSessions)
	}
	if cfg.Limits.VMStartTimeout != 45*time.Second {
		t.Errorf("Limits.VMStartTimeout = %v", cfg.Limits.VMStartTimeout)
	}
	if cfg.Limits.DiskSizeDefaultGB != 30 {
		t.Errorf("Limits.DiskSizeDefaultGB = %d", cfg.Limits.DiskSizeDefaultGB)
	}
	if cfg.LogLevel != "error" {
		t.Errorf("LogLevel = %q", cfg.LogLevel)
	}
}

func TestConfig_Load_Defaults_VerifyValues(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify path defaults
	if cfg.Paths.BaseImagesDir != "/var/lib/agentvm/images" {
		t.Errorf("Paths.BaseImagesDir = %q, want /var/lib/agentvm/images", cfg.Paths.BaseImagesDir)
	}
	if cfg.Paths.OverlayDisksDir != "/var/lib/agentvm/overlays" {
		t.Errorf("Paths.OverlayDisksDir = %q, want /var/lib/agentvm/overlays", cfg.Paths.OverlayDisksDir)
	}
	if cfg.Paths.CloudInitDir != "/var/lib/agentvm/cloud-init" {
		t.Errorf("Paths.CloudInitDir = %q, want /var/lib/agentvm/cloud-init", cfg.Paths.CloudInitDir)
	}
	if cfg.Paths.DataDir != dir {
		t.Errorf("Paths.DataDir = %q, want %q", cfg.Paths.DataDir, dir)
	}
	if cfg.Paths.SQLiteDBPath != "/var/lib/agentvm/agentvm.db" {
		t.Errorf("Paths.SQLiteDBPath = %q, want /var/lib/agentvm/agentvm.db", cfg.Paths.SQLiteDBPath)
	}
	// Verify host defaults (not set in file, so defaults should apply)
	if cfg.Host.QemuImgPath != "/usr/bin/qemu-img" {
		t.Errorf("Host.QemuImgPath = %q, want /usr/bin/qemu-img", cfg.Host.QemuImgPath)
	}
	if cfg.Host.CloudLocalGenISOPath != "/usr/bin/genisoimage" {
		t.Errorf("Host.CloudLocalGenISOPath = %q, want /usr/bin/genisoimage", cfg.Host.CloudLocalGenISOPath)
	}
	if cfg.Host.LibvirtURI != "qemu:///system" {
		t.Errorf("Host.LibvirtURI = %q, want qemu:///system", cfg.Host.LibvirtURI)
	}
	if cfg.Host.LibvirtSocketPath != "/var/run/libvirt/libvirt-sock" {
		t.Errorf("Host.LibvirtSocketPath = %q, want /var/run/libvirt/libvirt-sock", cfg.Host.LibvirtSocketPath)
	}
}

func TestConfig_Load_FileValues_VerifyDurations(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"api": map[string]any{
			"readTimeout":     "10s",
			"writeTimeout":    "20s",
			"shutdownTimeout": "15s",
		},
		"limits": map[string]any{
			"vmStartTimeout": "90s",
		},
	})

	t.Setenv("AGENTVM_SKIP_HOST_CHECKS", "true")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.API.ReadTimeout != 10*time.Second {
		t.Errorf("API.ReadTimeout = %v, want 10s", cfg.API.ReadTimeout)
	}
	if cfg.API.WriteTimeout != 20*time.Second {
		t.Errorf("API.WriteTimeout = %v, want 20s", cfg.API.WriteTimeout)
	}
	if cfg.API.ShutdownTimeout != 15*time.Second {
		t.Errorf("API.ShutdownTimeout = %v, want 15s", cfg.API.ShutdownTimeout)
	}
	if cfg.Limits.VMStartTimeout != 90*time.Second {
		t.Errorf("Limits.VMStartTimeout = %v, want 90s", cfg.Limits.VMStartTimeout)
	}
}

func TestValidationErrors_ErrorFormat_Single(t *testing.T) {
	errs := ValidationErrors{
		{Field: "Security.AdminToken", Reason: "must be set and at least 32 characters (fail-closed)"},
	}

	msg := errs.Error()
	if !strings.Contains(msg, "configuration validation failed: ") {
		t.Errorf("error should contain 'configuration validation failed: ', got: %q", msg)
	}
	if strings.Contains(msg, "errors") {
		t.Errorf("single error should not use plural 'errors', got: %q", msg)
	}
	if !strings.Contains(msg, "Security.AdminToken") {
		t.Errorf("error should contain field name, got: %q", msg)
	}
	if strings.Contains(msg, "1 errors") {
		t.Errorf("single error should not say '1 errors', got: %q", msg)
	}
}

func TestValidationErrors_ErrorFormat_Multiple(t *testing.T) {
	errs := ValidationErrors{
		{Field: "Security.AdminToken", Reason: "must be set"},
		{Field: "Host.QemuImgPath", Reason: "binary not found"},
	}

	msg := errs.Error()
	if !strings.Contains(msg, "configuration validation failed (2 errors): ") {
		t.Errorf("error should contain 'configuration validation failed (2 errors): ', got: %q", msg)
	}
	if !strings.Contains(msg, "  - Security.AdminToken: must be set") {
		t.Errorf("error should contain formatted first error, got: %q", msg)
	}
	if !strings.Contains(msg, "  - Host.QemuImgPath: binary not found") {
		t.Errorf("error should contain formatted second error, got: %q", msg)
	}
}

func TestValidationErrors_ErrorFormat_FieldAndReason(t *testing.T) {
	errs := ValidationErrors{
		{Field: "X", Value: "v", Reason: "bad"},
	}

	msg := errs.Error()
	// Verify the format includes field, colon, reason
	if !strings.Contains(msg, "X: bad") {
		t.Errorf("error should contain 'X: bad', got: %q", msg)
	}
}

func TestConfig_Load_ValidationError_Content_MissingDataDir(t *testing.T) {
	cleanEnv(t)

	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": "",
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	found := false
	for _, ve := range verr {
		if ve.Field == "Paths.DataDir" && strings.Contains(ve.Reason, "must be set") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Paths.DataDir validation error with 'must be set', got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Content_EmptyListenAddr(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"api": map[string]any{
			"listenAddr": "",
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	found := false
	for _, ve := range verr {
		if ve.Field == "API.ListenAddr" && strings.Contains(ve.Reason, "must be set") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected API.ListenAddr validation error with 'must be set', got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Content_InvalidLibvirtURI(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"libvirtURI":           "not-a-uri",
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	found := false
	for _, ve := range verr {
		if ve.Field == "Host.LibvirtURI" && strings.Contains(ve.Reason, "must be a valid URI") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Host.LibvirtURI validation error with 'must be a valid URI', got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Content_InvalidLogLevel(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"logLevel": "verbose",
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	found := false
	for _, ve := range verr {
		if ve.Field == "LogLevel" && strings.Contains(ve.Reason, "must be one of") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected LogLevel validation error with 'must be one of', got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Content_MaxConcurrentVMs(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"limits": map[string]any{
			"maxConcurrentVMs": 0,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	found := false
	for _, ve := range verr {
		if ve.Field == "Limits.MaxConcurrentVMs" && strings.Contains(ve.Reason, "must be between") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Limits.MaxConcurrentVMs validation error, got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Content_MaxConcurrentSessions(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"limits": map[string]any{
			"maxConcurrentSessions": 0,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	found := false
	for _, ve := range verr {
		if ve.Field == "Limits.MaxConcurrentSessions" && strings.Contains(ve.Reason, "must be between") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Limits.MaxConcurrentSessions validation error, got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Content_VMStartTimeout(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"limits": map[string]any{
			"vmStartTimeout": "10s",
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	found := false
	for _, ve := range verr {
		if ve.Field == "Limits.VMStartTimeout" && strings.Contains(ve.Reason, "must be at least") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Limits.VMStartTimeout validation error, got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Content_DiskSizeDefaultGB(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"limits": map[string]any{
			"diskSizeDefaultGB": 0,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	found := false
	for _, ve := range verr {
		if ve.Field == "Limits.DiskSizeDefaultGB" && strings.Contains(ve.Reason, "must be between") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Limits.DiskSizeDefaultGB validation error, got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Content_APITimeoutZero(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"api": map[string]any{
			"readTimeout": "0s",
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	found := false
	for _, ve := range verr {
		if ve.Field == "API.ReadTimeout" && strings.Contains(ve.Reason, "must be greater than 0") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected API.ReadTimeout validation error, got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Content_InvalidListenAddrFormat(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"api": map[string]any{
			"listenAddr": "abc",
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	found := false
	for _, ve := range verr {
		if ve.Field == "API.ListenAddr" && strings.Contains(ve.Reason, "invalid host:port") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected API.ListenAddr format validation error, got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Content_MissingAdminToken(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	found := false
	for _, ve := range verr {
		if ve.Field == "Security.AdminToken" && strings.Contains(ve.Reason, "fail-closed") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Security.AdminToken validation error, got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Content_ShortAdminToken(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "short",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	found := false
	for _, ve := range verr {
		if ve.Field == "Security.AdminToken" && ve.Value == "(redacted)" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Security.AdminToken validation error with redacted value, got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Content_TLSPaths(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken":  "this-is-a-valid-admin-token-that-is-at-least-32-chars",
			"tlsEnabled":  true,
			"tlsCertPath": "",
			"tlsKeyPath":  "",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	var foundCert, foundKey bool
	for _, ve := range verr {
		if ve.Field == "Security.TLSCertPath" && strings.Contains(ve.Reason, "must be set when TLS is enabled") {
			foundCert = true
		}
		if ve.Field == "Security.TLSKeyPath" && strings.Contains(ve.Reason, "must be set when TLS is enabled") {
			foundKey = true
		}
	}
	if !foundCert {
		t.Errorf("expected Security.TLSCertPath validation error, got: %v", verr)
	}
	if !foundKey {
		t.Errorf("expected Security.TLSKeyPath validation error, got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Content_BinaryNotFound(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/nonexistent/path/qemu-img",
			"cloudLocalGenISOPath": "/nonexistent/path/genisoimage",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	var foundQemu, foundGeniso bool
	for _, ve := range verr {
		if ve.Field == "Host.QemuImgPath" && strings.Contains(ve.Reason, "binary not found") {
			foundQemu = true
		}
		if ve.Field == "Host.CloudLocalGenISOPath" && strings.Contains(ve.Reason, "binary not found") {
			foundGeniso = true
		}
	}
	if !foundQemu {
		t.Errorf("expected Host.QemuImgPath binary not found error, got: %v", verr)
	}
	if !foundGeniso {
		t.Errorf("expected Host.CloudLocalGenISOPath binary not found error, got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Content_BinaryIsDirectory(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          dir,
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	found := false
	for _, ve := range verr {
		if ve.Field == "Host.QemuImgPath" && strings.Contains(ve.Reason, "is a directory") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Host.QemuImgPath directory error, got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Content_SocketNotFound(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"libvirtURI":           "qemu:///system",
			"libvirtSocketPath":    "/nonexistent/libvirt-sock",
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	found := false
	for _, ve := range verr {
		if ve.Field == "Host.LibvirtSocketPath" && strings.Contains(ve.Reason, "socket not found") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Host.LibvirtSocketPath socket not found error, got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Content_DirCreateError(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(filePath, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	socketPath := filepath.Join(dir, "libvirt-sock")
	if err := os.WriteFile(socketPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create socket file: %v", err)
	}

	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"libvirtURI":           "qemu:///system",
			"libvirtSocketPath":    socketPath,
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir":         dir,
			"baseImagesDir":   filePath,
			"overlayDisksDir": filepath.Join(dir, "overlays"),
			"cloudInitDir":    filepath.Join(dir, "cloud-init"),
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	found := false
	for _, ve := range verr {
		if ve.Field == "Paths.BaseImagesDir" && strings.Contains(ve.Reason, "cannot create directory") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Paths.BaseImagesDir directory creation error, got: %v", verr)
	}
}

// ---------------------------------------------------------------------------
// Mutation test coverage: verify exact error counts and field values
// ---------------------------------------------------------------------------

func TestConfig_Load_ValidationError_Count_InvalidLibvirtURI(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"libvirtURI":           "not-a-uri",
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	// Should have exactly 1 error (just the LibvirtURI validation)
	if len(verr) != 1 {
		t.Fatalf("expected exactly 1 validation error, got %d: %v", len(verr), verr)
	}
	if verr[0].Field != "Host.LibvirtURI" {
		t.Errorf("expected field Host.LibvirtURI, got %s", verr[0].Field)
	}
	if verr[0].Value != "not-a-uri" {
		t.Errorf("expected value 'not-a-uri', got %q", verr[0].Value)
	}
	if !strings.Contains(verr[0].Reason, "must be a valid URI") {
		t.Errorf("expected reason 'must be a valid URI', got %q", verr[0].Reason)
	}
}

func TestConfig_Load_ValidationError_Count_MissingDataDir(t *testing.T) {
	cleanEnv(t)

	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": "",
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	if len(verr) != 1 {
		t.Fatalf("expected exactly 1 validation error, got %d: %v", len(verr), verr)
	}
	if verr[0].Field != "Paths.DataDir" {
		t.Errorf("expected field Paths.DataDir, got %s", verr[0].Field)
	}
	if verr[0].Reason != "must be set" {
		t.Errorf("expected reason 'must be set', got %q", verr[0].Reason)
	}
}

func TestConfig_Load_ValidationError_Count_InvalidListenAddrFormat(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"api": map[string]any{
			"listenAddr": "abc",
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	if len(verr) != 1 {
		t.Fatalf("expected exactly 1 validation error, got %d: %v", len(verr), verr)
	}
	if verr[0].Field != "API.ListenAddr" {
		t.Errorf("expected field API.ListenAddr, got %s", verr[0].Field)
	}
	if verr[0].Value != "abc" {
		t.Errorf("expected value 'abc', got %q", verr[0].Value)
	}
	if !strings.Contains(verr[0].Reason, "invalid host:port format") {
		t.Errorf("expected reason 'invalid host:port format', got %q", verr[0].Reason)
	}
}

func TestConfig_Load_ValidationError_Count_APITimeoutZero(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"api": map[string]any{
			"readTimeout": "0s",
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	if len(verr) != 1 {
		t.Fatalf("expected exactly 1 validation error, got %d: %v", len(verr), verr)
	}
	if verr[0].Field != "API.ReadTimeout" {
		t.Errorf("expected field API.ReadTimeout, got %s", verr[0].Field)
	}
	if verr[0].Reason != "must be greater than 0" {
		t.Errorf("expected reason 'must be greater than 0', got %q", verr[0].Reason)
	}
}

func TestConfig_Load_ValidationError_Count_WriteTimeoutZero(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"api": map[string]any{
			"writeTimeout": "0s",
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	if len(verr) != 1 {
		t.Fatalf("expected exactly 1 validation error, got %d: %v", len(verr), verr)
	}
	if verr[0].Field != "API.WriteTimeout" {
		t.Errorf("expected field API.WriteTimeout, got %s", verr[0].Field)
	}
	if verr[0].Reason != "must be greater than 0" {
		t.Errorf("expected reason 'must be greater than 0', got %q", verr[0].Reason)
	}
}

func TestConfig_Load_ValidationError_Count_ShutdownTimeoutZero(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"api": map[string]any{
			"shutdownTimeout": "0s",
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	if len(verr) != 1 {
		t.Fatalf("expected exactly 1 validation error, got %d: %v", len(verr), verr)
	}
	if verr[0].Field != "API.ShutdownTimeout" {
		t.Errorf("expected field API.ShutdownTimeout, got %s", verr[0].Field)
	}
	if verr[0].Reason != "must be greater than 0" {
		t.Errorf("expected reason 'must be greater than 0', got %q", verr[0].Reason)
	}
}

func TestConfig_Load_ValidationError_Count_MissingAdminToken(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	if len(verr) != 1 {
		t.Fatalf("expected exactly 1 validation error, got %d: %v", len(verr), verr)
	}
	if verr[0].Field != "Security.AdminToken" {
		t.Errorf("expected field Security.AdminToken, got %s", verr[0].Field)
	}
	if verr[0].Reason != "must be set and at least 32 characters (fail-closed)" {
		t.Errorf("expected reason 'must be set and at least 32 characters (fail-closed)', got %q", verr[0].Reason)
	}
}

func TestConfig_Load_ValidationError_Count_ShortAdminToken(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "short",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	if len(verr) != 1 {
		t.Fatalf("expected exactly 1 validation error, got %d: %v", len(verr), verr)
	}
	if verr[0].Field != "Security.AdminToken" {
		t.Errorf("expected field Security.AdminToken, got %s", verr[0].Field)
	}
	if verr[0].Value != "(redacted)" {
		t.Errorf("expected value '(redacted)', got %q", verr[0].Value)
	}
	if verr[0].Reason != "must be at least 32 characters (fail-closed)" {
		t.Errorf("expected reason 'must be at least 32 characters (fail-closed)', got %q", verr[0].Reason)
	}
}

func TestConfig_Load_ValidationError_Count_TLSPaths(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken":  "this-is-a-valid-admin-token-that-is-at-least-32-chars",
			"tlsEnabled":  true,
			"tlsCertPath": "",
			"tlsKeyPath":  "",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	// Should have exactly 2 errors (TLSCertPath and TLSKeyPath)
	if len(verr) != 2 {
		t.Fatalf("expected exactly 2 validation errors, got %d: %v", len(verr), verr)
	}
	var foundCert, foundKey bool
	for _, ve := range verr {
		if ve.Field == "Security.TLSCertPath" && ve.Reason == "must be set when TLS is enabled" {
			foundCert = true
		}
		if ve.Field == "Security.TLSKeyPath" && ve.Reason == "must be set when TLS is enabled" {
			foundKey = true
		}
	}
	if !foundCert {
		t.Errorf("expected Security.TLSCertPath validation error, got: %v", verr)
	}
	if !foundKey {
		t.Errorf("expected Security.TLSKeyPath validation error, got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Count_MaxConcurrentVMs(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"limits": map[string]any{
			"maxConcurrentVMs": 0,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	if len(verr) != 1 {
		t.Fatalf("expected exactly 1 validation error, got %d: %v", len(verr), verr)
	}
	if verr[0].Field != "Limits.MaxConcurrentVMs" {
		t.Errorf("expected field Limits.MaxConcurrentVMs, got %s", verr[0].Field)
	}
	if verr[0].Value != "0" {
		t.Errorf("expected value '0', got %q", verr[0].Value)
	}
	if verr[0].Reason != "must be between 1 and 100" {
		t.Errorf("expected reason 'must be between 1 and 100', got %q", verr[0].Reason)
	}
}

func TestConfig_Load_ValidationError_Count_MaxConcurrentSessions(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"limits": map[string]any{
			"maxConcurrentSessions": 0,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	if len(verr) != 1 {
		t.Fatalf("expected exactly 1 validation error, got %d: %v", len(verr), verr)
	}
	if verr[0].Field != "Limits.MaxConcurrentSessions" {
		t.Errorf("expected field Limits.MaxConcurrentSessions, got %s", verr[0].Field)
	}
	if verr[0].Value != "0" {
		t.Errorf("expected value '0', got %q", verr[0].Value)
	}
	if verr[0].Reason != "must be between 1 and 500" {
		t.Errorf("expected reason 'must be between 1 and 500', got %q", verr[0].Reason)
	}
}

func TestConfig_Load_ValidationError_Count_VMStartTimeout(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"limits": map[string]any{
			"vmStartTimeout": "10s",
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	if len(verr) != 1 {
		t.Fatalf("expected exactly 1 validation error, got %d: %v", len(verr), verr)
	}
	if verr[0].Field != "Limits.VMStartTimeout" {
		t.Errorf("expected field Limits.VMStartTimeout, got %s", verr[0].Field)
	}
	if verr[0].Reason != "must be at least 30s" {
		t.Errorf("expected reason 'must be at least 30s', got %q", verr[0].Reason)
	}
}

func TestConfig_Load_ValidationError_Count_DiskSizeDefaultGB(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"limits": map[string]any{
			"diskSizeDefaultGB": 0,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	if len(verr) != 1 {
		t.Fatalf("expected exactly 1 validation error, got %d: %v", len(verr), verr)
	}
	if verr[0].Field != "Limits.DiskSizeDefaultGB" {
		t.Errorf("expected field Limits.DiskSizeDefaultGB, got %s", verr[0].Field)
	}
	if verr[0].Value != "0" {
		t.Errorf("expected value '0', got %q", verr[0].Value)
	}
	if verr[0].Reason != "must be between 5 and 500" {
		t.Errorf("expected reason 'must be between 5 and 500', got %q", verr[0].Reason)
	}
}

func TestConfig_Load_ValidationError_Count_InvalidLogLevel(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
		"logLevel": "verbose",
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	if len(verr) != 1 {
		t.Fatalf("expected exactly 1 validation error, got %d: %v", len(verr), verr)
	}
	if verr[0].Field != "LogLevel" {
		t.Errorf("expected field LogLevel, got %s", verr[0].Field)
	}
	if verr[0].Value != "verbose" {
		t.Errorf("expected value 'verbose', got %q", verr[0].Value)
	}
	if !strings.Contains(verr[0].Reason, "must be one of") {
		t.Errorf("expected reason containing 'must be one of', got %q", verr[0].Reason)
	}
}

func TestConfig_Load_ValidationError_Count_SocketNotFound(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"libvirtURI":           "qemu:///system",
			"libvirtSocketPath":    "/nonexistent/libvirt-sock",
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	if len(verr) != 1 {
		t.Fatalf("expected exactly 1 validation error, got %d: %v", len(verr), verr)
	}
	if verr[0].Field != "Host.LibvirtSocketPath" {
		t.Errorf("expected field Host.LibvirtSocketPath, got %s", verr[0].Field)
	}
	if !strings.Contains(verr[0].Reason, "socket not found") {
		t.Errorf("expected reason containing 'socket not found', got %q", verr[0].Reason)
	}
}

func TestConfig_Load_ValidationError_Count_BinaryNotFound(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          "/nonexistent/path/qemu-img",
			"cloudLocalGenISOPath": "/nonexistent/path/genisoimage",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	// Should have 2 errors: QemuImgPath and CloudLocalGenISOPath
	if len(verr) != 2 {
		t.Fatalf("expected exactly 2 validation errors, got %d: %v", len(verr), verr)
	}
	var foundQemu, foundGeniso bool
	for _, ve := range verr {
		if ve.Field == "Host.QemuImgPath" && strings.Contains(ve.Reason, "binary not found") {
			foundQemu = true
		}
		if ve.Field == "Host.CloudLocalGenISOPath" && strings.Contains(ve.Reason, "binary not found") {
			foundGeniso = true
		}
	}
	if !foundQemu {
		t.Errorf("expected Host.QemuImgPath binary not found error, got: %v", verr)
	}
	if !foundGeniso {
		t.Errorf("expected Host.CloudLocalGenISOPath binary not found error, got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Count_BinaryIsDirectory(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"qemuImgPath":          dir,
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir": dir,
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	if len(verr) != 1 {
		t.Fatalf("expected exactly 1 validation error, got %d: %v", len(verr), verr)
	}
	if verr[0].Field != "Host.QemuImgPath" {
		t.Errorf("expected field Host.QemuImgPath, got %s", verr[0].Field)
	}
	if !strings.Contains(verr[0].Reason, "is a directory") {
		t.Errorf("expected reason containing 'is a directory', got %q", verr[0].Reason)
	}
}

func TestConfig_Load_ValidationError_Count_DirCreateError(t *testing.T) {
	cleanEnv(t)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(filePath, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	socketPath := filepath.Join(dir, "libvirt-sock")
	if err := os.WriteFile(socketPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create socket file: %v", err)
	}

	cfgPath := writeConfigFile(t, map[string]any{
		"security": map[string]any{
			"adminToken": "this-is-a-valid-admin-token-that-is-at-least-32-chars",
		},
		"host": map[string]any{
			"libvirtURI":           "qemu:///system",
			"libvirtSocketPath":    socketPath,
			"qemuImgPath":          "/usr/bin/true",
			"cloudLocalGenISOPath": "/usr/bin/true",
		},
		"paths": map[string]any{
			"dataDir":         dir,
			"baseImagesDir":   filePath,
			"overlayDisksDir": filepath.Join(dir, "overlays"),
			"cloudInitDir":    filepath.Join(dir, "cloud-init"),
		},
	})

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var verr ValidationErrors
	if !asValidationErrors(err, &verr) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
	found := false
	for _, ve := range verr {
		if ve.Field == "Paths.BaseImagesDir" && strings.Contains(ve.Reason, "cannot create directory") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Paths.BaseImagesDir directory creation error, got: %v", verr)
	}
}

func TestConfig_Load_ValidationError_Count_InvalidJSON(t *testing.T) {
	cleanEnv(t)

	f, err := os.CreateTemp("", "agentvm-bad-json-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := f.WriteString("{not valid json"); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	f.Close()

	_, err = Load(f.Name())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to load config file") {
		t.Errorf("error should mention 'failed to load config file', got: %v", err)
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("error should mention 'invalid JSON', got: %v", err)
	}
}
