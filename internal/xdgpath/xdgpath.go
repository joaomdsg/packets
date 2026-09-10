// Package xdgpath resolves the packets config and data directories per the
// XDG base directory spec.
package xdgpath

import (
	"fmt"
	"os"
	"path/filepath"
)

// ConfigDir returns ~/.config/packets, honoring $XDG_CONFIG_HOME.
func ConfigDir() (string, error) {
	return resolve("XDG_CONFIG_HOME", ".config")
}

// DataDir returns ~/.local/share/packets, honoring $XDG_DATA_HOME.
func DataDir() (string, error) {
	return resolve("XDG_DATA_HOME", filepath.Join(".local", "share"))
}

func resolve(envVar, fallbackRel string) (string, error) {
	base := os.Getenv(envVar)
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("xdgpath: resolve home dir: %s", err)
		}
		base = filepath.Join(home, fallbackRel)
	}
	return filepath.Join(base, "packets"), nil
}

// EnsureDir creates path (and parents) with mode 0700 if it does not exist.
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("xdgpath: create dir %s: %s", path, err)
	}
	return nil
}
