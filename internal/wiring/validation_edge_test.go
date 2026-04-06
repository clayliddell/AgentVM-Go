package wiring

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func validValidationConfig(t tb) *Config {
	t.Helper()
	dir := t.TempDir()
	return &Config{
		Host: HostConfig{
			LibvirtURI:           "qemu:///system",
			LibvirtSocketPath:    filepath.Join(dir, "libvirt-sock"),
			QemuImgPath:          "/usr/bin/true",
			CloudLocalGenISOPath: "/usr/bin/true",
		},
		Paths: PathConfig{
			BaseImagesDir:   filepath.Join(dir, "images"),
			OverlayDisksDir: filepath.Join(dir, "overlays"),
			CloudInitDir:    filepath.Join(dir, "cloud-init"),
			DataDir:         dir,
			SQLiteDBPath:    filepath.Join(dir, "agentvm.db"),
		},
		API: APIConfig{
			ListenAddr:      "127.0.0.1:8080",
			ReadTimeout:     time.Second,
			WriteTimeout:    2 * time.Second,
			ShutdownTimeout: 3 * time.Second,
		},
		Security: SecurityConfig{
			AdminToken: strings.Repeat("a", 32),
		},
		Limits: LimitsConfig{
			MaxConcurrentVMs:      1,
			MaxConcurrentSessions: 1,
			VMStartTimeout:        30 * time.Second,
			DiskSizeDefaultGB:     5,
		},
		LogLevel:       "info",
		SkipHostChecks: true,
	}
}

func TestValidate_BoundaryValues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{
			name: "admin-token-32-bytes",
			mutate: func(cfg *Config) {
				cfg.Security.AdminToken = strings.Repeat("b", 32)
			},
		},
		{
			name: "max-concurrent-vms-min",
			mutate: func(cfg *Config) {
				cfg.Limits.MaxConcurrentVMs = 1
			},
		},
		{
			name: "max-concurrent-vms-max",
			mutate: func(cfg *Config) {
				cfg.Limits.MaxConcurrentVMs = 100
			},
		},
		{
			name: "max-concurrent-sessions-min",
			mutate: func(cfg *Config) {
				cfg.Limits.MaxConcurrentSessions = 1
			},
		},
		{
			name: "max-concurrent-sessions-max",
			mutate: func(cfg *Config) {
				cfg.Limits.MaxConcurrentSessions = 500
			},
		},
		{
			name: "vm-start-timeout-min",
			mutate: func(cfg *Config) {
				cfg.Limits.VMStartTimeout = 30 * time.Second
			},
		},
		{
			name: "disk-size-min",
			mutate: func(cfg *Config) {
				cfg.Limits.DiskSizeDefaultGB = 5
			},
		},
		{
			name: "disk-size-max",
			mutate: func(cfg *Config) {
				cfg.Limits.DiskSizeDefaultGB = 500
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validValidationConfig(t)
			tt.mutate(cfg)

			errs := validate(cfg)
			if len(errs) != 0 {
				t.Fatalf("expected boundary config to validate, got %v", errs)
			}
		})
	}
}

func TestValidate_SkipsHostChecksOnEarlierErrors(t *testing.T) {
	cfg := validValidationConfig(t)
	cfg.API.ReadTimeout = 0
	cfg.Host.QemuImgPath = "/definitely/missing/qemu-img"
	cfg.Host.CloudLocalGenISOPath = "/definitely/missing/genisoimage"
	cfg.Host.LibvirtSocketPath = "/definitely/missing/libvirt-sock"
	cfg.SkipHostChecks = false

	errs := validate(cfg)
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 validation error before host checks, got %d: %v", len(errs), errs)
	}
	if errs[0].Field != "API.ReadTimeout" {
		t.Fatalf("expected API.ReadTimeout error, got %s", errs[0].Field)
	}
}

func TestValidationErrors_ErrorOutputExactFormatting(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		errs := ValidationErrors{{Field: "Security.AdminToken", Reason: "must be set and at least 32 characters (fail-closed)"}}
		want := "configuration validation failed: \n  - Security.AdminToken: must be set and at least 32 characters (fail-closed)"
		if got := errs.Error(); got != want {
			t.Fatalf("unexpected single-error formatting:\nwant: %q\n got: %q", want, got)
		}
	})

	t.Run("multiple", func(t *testing.T) {
		errs := ValidationErrors{
			{Field: "Security.AdminToken", Reason: "must be set and at least 32 characters (fail-closed)"},
			{Field: "Host.QemuImgPath", Reason: "binary not found"},
		}
		want := "configuration validation failed (2 errors): \n  - Security.AdminToken: must be set and at least 32 characters (fail-closed)\n  - Host.QemuImgPath: binary not found"
		if got := errs.Error(); got != want {
			t.Fatalf("unexpected multi-error formatting:\nwant: %q\n got: %q", want, got)
		}
	})
}
