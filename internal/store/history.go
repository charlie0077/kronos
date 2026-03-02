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

		for k != nil && len(records) < limit {
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
		for k, v := c.Last(); k != nil && len(records) < limit; k, v = c.Prev() {
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
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(runsBucket)
		c := b.Cursor()
		prefix := jobName + "/"

		// Collect all keys for this job.
		var keys [][]byte
		for k, _ := c.Seek([]byte(prefix)); k != nil && strings.HasPrefix(string(k), prefix); k, _ = c.Next() {
			keyCopy := make([]byte, len(k))
			copy(keyCopy, k)
			keys = append(keys, keyCopy)
		}

		// Delete oldest keys (keys are sorted chronologically).
		if len(keys) > keepN {
			for _, k := range keys[:len(keys)-keepN] {
				if err := b.Delete(k); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
