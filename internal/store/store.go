package store

import (
	"fmt"
	"os"
	"path/filepath"

	bolt "go.etcd.io/bbolt"
)

var runsBucket = []byte("runs")

// Store wraps a bbolt database for run history persistence.
type Store struct {
	db *bolt.DB
}

// Open creates or opens a bbolt database at the given path.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	db, err := bolt.Open(path, 0o600, nil)
	if err != nil {
		return nil, fmt.Errorf("opening database %s: %w", path, err)
	}

	// Ensure buckets exist.
	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(runsBucket)
		return err
	}); err != nil {
		db.Close()
		return nil, fmt.Errorf("initializing buckets: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}
