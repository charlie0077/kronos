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

	var data []byte
	if node != nil {
		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		if err := enc.Encode(node); err != nil {
			return fmt.Errorf("encoding yaml: %w", err)
		}
		enc.Close()
		data = buf.Bytes()
	} else {
		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		if err := enc.Encode(cfg); err != nil {
			return fmt.Errorf("encoding yaml: %w", err)
		}
		enc.Close()
		data = buf.Bytes()
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
