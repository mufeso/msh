package history

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"example.com/msh/internal/output"
)

// ErrNotFound means no record matches.
var ErrNotFound = errors.New("no such run in the history")

// Store keeps records as one JSON file each in Dir. A file per run means
// saving never touches existing data.
type Store struct {
	Dir string
}

func (s Store) path(id string) string {
	return filepath.Join(s.Dir, id+".json")
}

// Save writes a record. Results can contain sensitive output, so only
// the owner can read the directory and the files.
func (s Store) Save(r Record) error {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	return output.WriteFileAtomic(s.path(r.ID), 0o600, func(w io.Writer) error {
		return json.NewEncoder(w).Encode(r)
	})
}

// Load reads one record by its exact ID.
func (s Store) Load(id string) (Record, error) {
	data, err := os.ReadFile(s.path(id))
	if errors.Is(err, os.ErrNotExist) {
		return Record{}, ErrNotFound
	}
	if err != nil {
		return Record{}, err
	}
	var r Record
	if err := json.Unmarshal(data, &r); err != nil {
		return Record{}, fmt.Errorf("history record %s: %w", id, err)
	}
	return r, nil
}

// ids returns all record IDs, oldest first. IDs sort by time.
func (s Store) ids() ([]string, error) {
	entries, err := os.ReadDir(s.Dir) // sorted by file name
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if id, ok := strings.CutSuffix(e.Name(), ".json"); ok && !strings.HasPrefix(id, ".") {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// All iterates over the records, newest first:
//
//	for r, err := range store.All() { ... }
//
// A record is only read when the loop reaches it, so breaking out of the
// loop early saves reading the rest.
func (s Store) All() iter.Seq2[Record, error] {
	return func(yield func(Record, error) bool) {
		ids, err := s.ids()
		if err != nil {
			yield(Record{}, err)
			return
		}
		for _, id := range slices.Backward(ids) {
			if !yield(s.Load(id)) {
				return
			}
		}
	}
}

// Find returns a record by "last", an ID, or a unique prefix of an ID.
func (s Store) Find(ref string) (Record, error) {
	ids, err := s.ids()
	if err != nil {
		return Record{}, err
	}
	if ref == "last" {
		if len(ids) == 0 {
			return Record{}, ErrNotFound
		}
		return s.Load(ids[len(ids)-1])
	}
	var matches []string
	for _, id := range ids {
		if strings.HasPrefix(id, ref) {
			matches = append(matches, id)
		}
	}
	switch len(matches) {
	case 0:
		return Record{}, ErrNotFound
	case 1:
		return s.Load(matches[0])
	default:
		return Record{}, fmt.Errorf("%q matches %d runs; use more of the ID", ref, len(matches))
	}
}

// Prune removes all but the newest keep records and returns how many it
// removed.
func (s Store) Prune(keep int) (int, error) {
	ids, err := s.ids()
	if err != nil {
		return 0, err
	}
	if len(ids) <= keep {
		return 0, nil
	}
	removed := 0
	for _, id := range ids[:len(ids)-keep] {
		err := os.Remove(s.path(id))
		if errors.Is(err, os.ErrNotExist) {
			continue // another msh run removed it first
		}
		if err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}
