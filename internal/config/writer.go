package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Save writes the config to disk. If node is non-nil it updates values
// in the existing AST to preserve comments; otherwise it does a full marshal.
func Save(path string, cfg *Config, node *yaml.Node) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	var target any = cfg
	if node != nil {
		target = node
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(target); err != nil {
		return fmt.Errorf("encoding yaml: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("closing yaml encoder: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
