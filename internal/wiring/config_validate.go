package wiring

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// validate runs all validators and returns aggregated errors.
func validate(cfg *Config) ValidationErrors {
	var errs ValidationErrors

	errs = append(errs, validateHost(cfg)...)
	errs = append(errs, validatePaths(cfg)...)
	errs = append(errs, validateAPI(cfg)...)
	errs = append(errs, validateSecurity(cfg)...)
	errs = append(errs, validateLimits(cfg)...)
	errs = append(errs, validateLogLevel(cfg)...)

	if len(errs) > 0 {
		return errs
	}

	// Only run host prerequisite checks if validation passed so far.
	if !cfg.SkipHostChecks {
		errs = append(errs, checkHostPrerequisites(cfg)...)
	}

	return errs
}

// ---------------------------------------------------------------------------
// Individual validators
// ---------------------------------------------------------------------------

func validateHost(cfg *Config) ValidationErrors {
	var errs ValidationErrors

	if cfg.Host.LibvirtURI == "" || !libvirtURIRegex.MatchString(cfg.Host.LibvirtURI) {
		errs = append(errs, ValidationError{
			Field:  "Host.LibvirtURI",
			Value:  cfg.Host.LibvirtURI,
			Reason: fmt.Sprintf("must be a valid URI, got %q", cfg.Host.LibvirtURI),
		})
	}

	return errs
}

func validatePaths(cfg *Config) ValidationErrors {
	var errs ValidationErrors

	if cfg.Paths.DataDir == "" {
		errs = append(errs, ValidationError{
			Field:  "Paths.DataDir",
			Reason: "must be set",
		})
	}

	return errs
}

func validateAPI(cfg *Config) ValidationErrors {
	var errs ValidationErrors

	if cfg.API.ListenAddr == "" {
		errs = append(errs, ValidationError{
			Field:  "API.ListenAddr",
			Reason: "must be set",
		})
	} else {
		// Validate host:port format.
		if _, _, err := net.SplitHostPort(cfg.API.ListenAddr); err != nil {
			errs = append(errs, ValidationError{
				Field:  "API.ListenAddr",
				Value:  cfg.API.ListenAddr,
				Reason: fmt.Sprintf("invalid host:port format %q", cfg.API.ListenAddr),
			})
		}
	}

	if cfg.API.ReadTimeout <= 0 {
		errs = append(errs, ValidationError{
			Field:  "API.ReadTimeout",
			Value:  cfg.API.ReadTimeout.String(),
			Reason: "must be greater than 0",
		})
	}
	if cfg.API.WriteTimeout <= 0 {
		errs = append(errs, ValidationError{
			Field:  "API.WriteTimeout",
			Value:  cfg.API.WriteTimeout.String(),
			Reason: "must be greater than 0",
		})
	}
	if cfg.API.ShutdownTimeout <= 0 {
		errs = append(errs, ValidationError{
			Field:  "API.ShutdownTimeout",
			Value:  cfg.API.ShutdownTimeout.String(),
			Reason: "must be greater than 0",
		})
	}

	return errs
}

func validateSecurity(cfg *Config) ValidationErrors {
	var errs ValidationErrors

	if cfg.Security.AdminToken == "" {
		errs = append(errs, ValidationError{
			Field:  "Security.AdminToken",
			Reason: "must be set and at least 32 characters (fail-closed)",
		})
	} else if len(cfg.Security.AdminToken) < 32 {
		errs = append(errs, ValidationError{
			Field:  "Security.AdminToken",
			Value:  "(redacted)",
			Reason: "must be at least 32 characters (fail-closed)",
		})
	}

	if cfg.Security.TLSEnabled {
		if cfg.Security.TLSCertPath == "" {
			errs = append(errs, ValidationError{
				Field:  "Security.TLSCertPath",
				Reason: "must be set when TLS is enabled",
			})
		}
		if cfg.Security.TLSKeyPath == "" {
			errs = append(errs, ValidationError{
				Field:  "Security.TLSKeyPath",
				Reason: "must be set when TLS is enabled",
			})
		}
	}

	return errs
}

func validateLimits(cfg *Config) ValidationErrors {
	var errs ValidationErrors

	if cfg.Limits.MaxConcurrentVMs <= 0 || cfg.Limits.MaxConcurrentVMs > 100 {
		errs = append(errs, ValidationError{
			Field:  "Limits.MaxConcurrentVMs",
			Value:  strconv.Itoa(cfg.Limits.MaxConcurrentVMs),
			Reason: "must be between 1 and 100",
		})
	}
	if cfg.Limits.MaxConcurrentSessions <= 0 || cfg.Limits.MaxConcurrentSessions > 500 {
		errs = append(errs, ValidationError{
			Field:  "Limits.MaxConcurrentSessions",
			Value:  strconv.Itoa(cfg.Limits.MaxConcurrentSessions),
			Reason: "must be between 1 and 500",
		})
	}
	if cfg.Limits.VMStartTimeout < 30*time.Second {
		errs = append(errs, ValidationError{
			Field:  "Limits.VMStartTimeout",
			Value:  cfg.Limits.VMStartTimeout.String(),
			Reason: "must be at least 30s",
		})
	}
	if cfg.Limits.DiskSizeDefaultGB < 5 || cfg.Limits.DiskSizeDefaultGB > 500 {
		errs = append(errs, ValidationError{
			Field:  "Limits.DiskSizeDefaultGB",
			Value:  strconv.Itoa(cfg.Limits.DiskSizeDefaultGB),
			Reason: "must be between 5 and 500",
		})
	}

	return errs
}

func validateLogLevel(cfg *Config) ValidationErrors {
	var errs ValidationErrors

	if !validLogLevels[cfg.LogLevel] {
		errs = append(errs, ValidationError{
			Field:  "LogLevel",
			Value:  cfg.LogLevel,
			Reason: fmt.Sprintf("must be one of: debug, info, warn, error; got %q", cfg.LogLevel),
		})
	}

	return errs
}

// ---------------------------------------------------------------------------
// Host prerequisite checks
// ---------------------------------------------------------------------------

// checkHostPrerequisites verifies that required host binaries, sockets,
// and directories exist or can be created.
func checkHostPrerequisites(cfg *Config) ValidationErrors {
	var errs ValidationErrors

	// Check binary paths exist and are executable.
	checkBinary(cfg.Host.QemuImgPath, "Host.QemuImgPath", &errs)
	checkBinary(cfg.Host.CloudLocalGenISOPath, "Host.CloudLocalGenISOPath", &errs)

	// Check socket path exists.
	if _, err := os.Stat(cfg.Host.LibvirtSocketPath); err != nil {
		errs = append(errs, ValidationError{
			Field:  "Host.LibvirtSocketPath",
			Value:  cfg.Host.LibvirtSocketPath,
			Reason: fmt.Sprintf("socket not found: %v", err),
		})
	}

	// Ensure data directories exist or can be created.
	for name, path := range map[string]string{
		"Paths.BaseImagesDir":   cfg.Paths.BaseImagesDir,
		"Paths.OverlayDisksDir": cfg.Paths.OverlayDisksDir,
		"Paths.CloudInitDir":    cfg.Paths.CloudInitDir,
	} {
		if err := ensureDir(path); err != nil {
			errs = append(errs, ValidationError{
				Field:  name,
				Value:  path,
				Reason: fmt.Sprintf("cannot create directory: %v", err),
			})
		}
	}

	// Verify SQLite parent directory is writable.
	dbDir := filepath.Dir(cfg.Paths.SQLiteDBPath)
	if err := ensureDir(dbDir); err != nil {
		errs = append(errs, ValidationError{
			Field:  "Paths.SQLiteDBPath",
			Value:  cfg.Paths.SQLiteDBPath,
			Reason: fmt.Sprintf("parent directory not writable: %v", err),
		})
	}

	return errs
}

// checkBinary verifies that a binary exists and is executable.
func checkBinary(path, field string, errs *ValidationErrors) {
	info, err := os.Stat(path)
	if err != nil {
		*errs = append(*errs, ValidationError{
			Field:  field,
			Value:  path,
			Reason: fmt.Sprintf("binary not found at %q", path),
		})
		return
	}
	// Check executable bit.
	if info.Mode()&0111 == 0 {
		*errs = append(*errs, ValidationError{
			Field:  field,
			Value:  path,
			Reason: fmt.Sprintf("binary at %q is not executable", path),
		})
	}
}

// ensureDir creates a directory if it doesn't exist.
func ensureDir(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return nil
		}
		return fmt.Errorf("path exists but is not a directory: %s", path)
	}
	if os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return err
}
