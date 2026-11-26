package config

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const (
	// Windows Run registry key for current user startup
	startupKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	appName        = "OneSnap"
)

// IsStartupEnabled checks if the app is set to run on Windows startup
func IsStartupEnabled() (bool, error) {
	return isStartupEnabled(startupKeyPath)
}

func isStartupEnabled(path string) (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE)
	if err == registry.ErrNotExist {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer key.Close()

	_, _, err = key.GetStringValue(appName)
	if err == registry.ErrNotExist {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// SetStartupEnabled enables or disables running the app on Windows startup
func SetStartupEnabled(enabled bool) error {
	if enabled {
		return enableStartup(startupKeyPath)
	}
	return disableStartup(startupKeyPath)
}

func enableStartup(path string) error {
	// Get executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks and get absolute path
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Create (or open) the Run key for writing and reading (for verification).
	// CreateKey instead of OpenKey because the Run key may be missing in
	// minimal user hives (e.g. CI service sessions); OpenKey would fail
	// with ERROR_FILE_NOT_FOUND there.
	key, _, err := registry.CreateKey(registry.CURRENT_USER, path, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return fmt.Errorf("failed to create registry key: %w", err)
	}
	defer key.Close()

	// Set value with path in quotes (for paths with spaces)
	quotedPath := fmt.Sprintf(`"%s"`, exePath)
	if err := key.SetStringValue(appName, quotedPath); err != nil {
		return fmt.Errorf("failed to set registry value: %w", err)
	}

	// Verify write by reading back the value
	val, _, err := key.GetStringValue(appName)
	if err != nil {
		return fmt.Errorf("failed to verify registry write: %w", err)
	}
	if val != quotedPath {
		return fmt.Errorf("registry verification failed: expected %s, got %s", quotedPath, val)
	}

	return nil
}

// SyncStartupPath updates the registry startup path to match current executable.
// This fixes two issues:
// 1. Duplicate app issue when user downloads new version to different location
// 2. Missing registry entry when config says enabled but registry was never written
// Call this on app startup when startup is enabled in config.
func SyncStartupPath() error {
	// Always enable startup to ensure registry matches config.
	// If registry already has correct path, this is a no-op.
	// If registry is missing or has wrong path, this fixes it.
	return enableStartup(startupKeyPath)
}

func disableStartup(path string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.SET_VALUE)
	if err != nil {
		// If key doesn't exist, autostart is already disabled
		if err == registry.ErrNotExist {
			return nil
		}
		return fmt.Errorf("failed to open registry: %w", err)
	}
	defer key.Close()

	// Delete the value (ignore error if not exists)
	err = key.DeleteValue(appName)
	if err == registry.ErrNotExist {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to delete registry value: %w", err)
	}
	return nil
}
