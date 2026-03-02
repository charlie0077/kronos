package config

import (
	"os"
	"path/filepath"
)

const appName = "kronos"

// ConfigDir returns the OS-native config directory for Kronos.
func ConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(dir, appName)
}

// CacheDir returns the OS-native cache directory for Kronos.
func CacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = filepath.Join(os.Getenv("HOME"), ".cache")
	}
	return filepath.Join(dir, appName)
}

// DefaultConfigPath returns the default config file path.
func DefaultConfigPath() string {
	return filepath.Join(ConfigDir(), "kronos.yaml")
}

// LogDir returns the log directory, using settings override or cache default.
func LogDir(s Settings) string {
	if s.LogDir != "" {
		return s.LogDir
	}
	return filepath.Join(CacheDir(), "logs")
}

// DBPath returns the path to the bbolt database.
func DBPath() string {
	return filepath.Join(CacheDir(), "kronos.db")
}

// PIDPath returns the path to the PID lock file.
func PIDPath() string {
	return filepath.Join(CacheDir(), "kronos.pid")
}

// EnsureDir creates a directory if it doesn't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
