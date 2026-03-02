package store

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

// RunRecord represents a single job execution.
type RunRecord struct {
	JobName   string    `json:"job_name"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	ExitCode  int       `json:"exit_code"`
	Output    string    `json:"output"`
	Trigger   string    `json:"trigger"` // "scheduled" or "manual"
	Success   bool      `json:"success"`
}

const keyTimeFmt = "2006-01-02T15:04:05.000Z"

// SaveRun persists a run record.
func (s *Store) SaveRun(record RunRecord) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(runsBucket)
		key := record.JobName + "/" + record.StartTime.UTC().Format(keyTimeFmt)
		data, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("marshaling run record: %w", err)
		}
		return b.Put([]byte(key), data)
	})
}

// GetRuns returns the most recent runs for a specific job, newest first.
func (s *Store) GetRuns(jobName string, limit int) ([]RunRecord, error) {
	var records []RunRecord
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(runsBucket)
		c := b.Cursor()
		prefix := jobName + "/"

		// Seek to end of prefix range and iterate backwards.
		// Find the last key with this prefix by seeking past it.
		end := []byte(jobName + "0") // '0' is after '/' in ASCII
		c.Seek(end)
		k, v := c.Prev()

		for k != nil && (limit <= 0 || len(records) < limit) {
			if !strings.HasPrefix(string(k), prefix) {
				break
			}
			var rec RunRecord
			if err := json.Unmarshal(v, &rec); err == nil {
				records = append(records, rec)
			}
			k, v = c.Prev()
		}
		return nil
	})
	return records, err
}

// GetAllRuns returns the most recent runs across all jobs, newest first.
func (s *Store) GetAllRuns(limit int) ([]RunRecord, error) {
	var records []RunRecord
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(runsBucket)
		c := b.Cursor()

		// Iterate from the end backwards.
		for k, v := c.Last(); k != nil && (limit <= 0 || len(records) < limit); k, v = c.Prev() {
			var rec RunRecord
			if err := json.Unmarshal(v, &rec); err == nil {
				records = append(records, rec)
			}
		}
		return nil
	})
	return records, err
}

// GetLastRun returns the most recent run for a job, or nil if none.
func (s *Store) GetLastRun(jobName string) (*RunRecord, error) {
	runs, err := s.GetRuns(jobName, 1)
	if err != nil || len(runs) == 0 {
		return nil, err
	}
	return &runs[0], nil
}

// PruneHistory keeps only the last keepN records per job, deleting older ones.
func (s *Store) PruneHistory(jobName string, keepN int) error {
	_, err := s.PruneKeepN(jobName, keepN)
	return err
}

// iterateOlderThan calls fn for each key whose run record is older than cutoff.
// When jobName is non-empty, only records for that job are visited (using prefix seek).
func iterateOlderThan(b *bolt.Bucket, cutoff time.Time, jobName string, fn func(k []byte)) {
	c := b.Cursor()

	var first func() ([]byte, []byte)
	var match func(k []byte) bool

	if jobName != "" {
		prefix := jobName + "/"
		first = func() ([]byte, []byte) { return c.Seek([]byte(prefix)) }
		match = func(k []byte) bool { return strings.HasPrefix(string(k), prefix) }
	} else {
		first = func() ([]byte, []byte) { return c.First() }
		match = func([]byte) bool { return true }
	}

	for k, v := first(); k != nil && match(k); k, v = c.Next() {
		var rec RunRecord
		if err := json.Unmarshal(v, &rec); err != nil {
			continue
		}
		if rec.StartTime.Before(cutoff) {
			fn(k)
		}
	}
}

// PruneOlderThan deletes run records older than cutoff. If jobName is non-empty,
// only records for that job are considered. Returns the count of deleted records.
func (s *Store) PruneOlderThan(cutoff time.Time, jobName string) (int, error) {
	deleted := 0
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(runsBucket)

		var keysToDelete [][]byte
		iterateOlderThan(b, cutoff, jobName, func(k []byte) {
			keyCopy := make([]byte, len(k))
			copy(keyCopy, k)
			keysToDelete = append(keysToDelete, keyCopy)
		})

		for _, k := range keysToDelete {
			if err := b.Delete(k); err != nil {
				return err
			}
			deleted++
		}
		return nil
	})
	return deleted, err
}

// CountOlderThan counts run records older than cutoff without deleting.
// If jobName is non-empty, only records for that job are counted.
func (s *Store) CountOlderThan(cutoff time.Time, jobName string) (int, error) {
	count := 0
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(runsBucket)
		iterateOlderThan(b, cutoff, jobName, func(_ []byte) {
			count++
		})
		return nil
	})
	return count, err
}

// PruneKeepN keeps only the most recent keepN records for a job, deleting
// older ones. Returns the count of deleted records.
func (s *Store) PruneKeepN(jobName string, keepN int) (int, error) {
	deleted := 0
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(runsBucket)
		c := b.Cursor()
		prefix := jobName + "/"

		var keys [][]byte
		for k, _ := c.Seek([]byte(prefix)); k != nil && strings.HasPrefix(string(k), prefix); k, _ = c.Next() {
			keyCopy := make([]byte, len(k))
			copy(keyCopy, k)
			keys = append(keys, keyCopy)
		}

		if len(keys) > keepN {
			for _, k := range keys[:len(keys)-keepN] {
				if err := b.Delete(k); err != nil {
					return err
				}
				deleted++
			}
		}
		return nil
	})
	return deleted, err
}

// CountPruneKeepN counts how many records would be deleted if keeping only
// the most recent keepN for a job.
func (s *Store) CountPruneKeepN(jobName string, keepN int) (int, error) {
	count := 0
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(runsBucket)
		c := b.Cursor()
		prefix := jobName + "/"

		total := 0
		for k, _ := c.Seek([]byte(prefix)); k != nil && strings.HasPrefix(string(k), prefix); k, _ = c.Next() {
			total++
		}

		if total > keepN {
			count = total - keepN
		}
		return nil
	})
	return count, err
}

// GetAllJobNames scans all keys in the runs bucket and returns sorted unique
// job name prefixes. It uses prefix-seeking to skip past all records for each
// discovered job instead of iterating one-by-one.
func (s *Store) GetAllJobNames() ([]string, error) {
	// BoltDB keys are sorted, so prefix-seeking yields names in order already.
	var names []string
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(runsBucket)
		c := b.Cursor()

		for k, _ := c.First(); k != nil; {
			key := string(k)
			idx := strings.Index(key, "/")
			if idx < 0 {
				k, _ = c.Next()
				continue
			}
			name := key[:idx]
			names = append(names, name)
			// Skip past all keys with this prefix by seeking to the next prefix.
			// '0' is the character after '/' in ASCII, so name+"0" is just past name+"/...".
			k, _ = c.Seek([]byte(name + "0"))
		}
		return nil
	})

	return names, err
}
