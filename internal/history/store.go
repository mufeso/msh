package history

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
