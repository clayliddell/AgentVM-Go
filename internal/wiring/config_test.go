package wiring

import (
	"encoding/json"
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
