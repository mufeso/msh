package runbook

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"example.com/msh/internal/output"
)

// Progress records how far each server got. The engine depends on this
// small interface, not on files.
type Progress interface {
	// Completed returns the number of finished steps for server.
	Completed(server string) int
	// Mark records that the first n steps of server are finished.
	Mark(server string, n int)
}

// state is the content of a state file.
type state struct {
	Runbook string         `json:"runbook"`
	Hash    string         `json:"plan_hash"`
	Done    map[string]int `json:"done"` // server -> completed steps
}

// Tracker stores progress in a file. It is safe for concurrent use.
type Tracker struct {
	path string

	mu    sync.Mutex
	state state
	err   error // first write error
}

// ErrNoState means there is nothing to resume.
var ErrNoState = errors.New("no saved progress to resume")

// PlanHash fingerprints a plan: the same steps and commands give the same
// hash, so changed runbooks and arguments are detected.
func PlanHash(plan []PlannedStep) string {
	h := sha256.New()
	for _, ps := range plan {
		fmt.Fprintf(h, "%q %q %x\n", ps.Step.Name, ps.Command, ps.Stdin)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// NewTracker returns a tracker for the state file at path. With resume,
// the saved state is loaded and must belong to this plan; otherwise any
// old state is replaced.
func NewTracker(path, runbook, hash string, resume bool) (*Tracker, error) {
	t := &Tracker{path: path, state: state{Runbook: runbook, Hash: hash, Done: map[string]int{}}}
	if !resume {
		return t, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNoState
	}
	if err != nil {
		return nil, err
	}
	var saved state
	if err := json.Unmarshal(data, &saved); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if saved.Hash != hash {
		return nil, errors.New("the runbook or its arguments changed since the saved run; run it again without --resume")
	}
	if saved.Done != nil {
		t.state.Done = saved.Done
	}
	return t, nil
}

// Completed implements Progress.
func (t *Tracker) Completed(server string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state.Done[server]
}

// Mark implements Progress. The file is written atomically after every
// step, so a crash loses at most the step that was running.
func (t *Tracker) Mark(server string, n int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state.Done[server] = n
	if err := t.save(); err != nil && t.err == nil {
		t.err = err
	}
}

func (t *Tracker) save() error {
	if err := os.MkdirAll(filepath.Dir(t.path), 0o700); err != nil {
		return err
	}
	return output.WriteFileAtomic(t.path, 0o600, func(w io.Writer) error {
		return json.NewEncoder(w).Encode(t.state)
	})
}

// Err returns the first error from saving progress.
func (t *Tracker) Err() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.err
}

// Finish removes the state file after a fully successful run.
func (t *Tracker) Finish() error {
	err := os.Remove(t.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
