package app

import (
	"fmt"
	"path/filepath"
	"time"

	"example.com/msh/internal/config"
	"example.com/msh/internal/history"
	"example.com/msh/internal/runner"
)

// historyStore returns the store. ok is false if no state directory can
// be determined.
func (c *cli) historyStore() (history.Store, bool) {
	dir := config.StateDir(c.lookupEnv)
	if dir == "" {
		return history.Store{}, false
	}
	return history.Store{Dir: filepath.Join(dir, "history")}, true
}

// recordHistory saves the run. It is best effort: the run has happened,
// so a full disk here only deserves a warning.
func (c *cli) recordHistory(opts *commonOptions, report runner.Report) {
	store, ok := c.historyStore()
	if !ok {
		return
	}
	if err := store.Save(history.NewRecord(c.args, currentUser(), report, time.Now())); err != nil {
		fmt.Fprintf(c.stderr, "msh: warning: could not save the run in the history: %v\n", err)
		return
	}
	c.log(opts).Debug("run recorded in the history")
	if opts.historyLimit > 0 {
		if _, err := store.Prune(opts.historyLimit); err != nil {
			fmt.Fprintf(c.stderr, "msh: warning: could not prune the history: %v\n", err)
		}
	}
}
