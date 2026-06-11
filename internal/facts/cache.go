package facts

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"

	"example.com/msh/internal/inventory"
	"example.com/msh/internal/output"
)

// entry is one server's cache file.
type entry struct {
	Address   string    `json:"address"`
	Collected time.Time `json:"collected"`
	Facts     Facts     `json:"facts"`
}

func (g *Gatherer) now() time.Time {
	if g.Now != nil {
		return g.Now()
	}
	return time.Now()
}

func (g *Gatherer) path(s inventory.Server) string {
	return filepath.Join(g.CacheDir, "facts", s.Name+".json")
}

// cached returns unexpired facts for s. Any problem with the cache just
// means the facts are collected again.
func (g *Gatherer) cached(s inventory.Server) (Facts, bool) {
	if g.CacheDir == "" || g.Refresh || g.TTL <= 0 {
		return nil, false
	}
	data, err := os.ReadFile(g.path(s))
	if err != nil {
		return nil, false
	}
	var e entry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, false
	}
	if e.Address != s.Address || g.now().Sub(e.Collected) > g.TTL {
		return nil, false
	}
	return e.Facts, true
}

// store saves facts. Failing to write the cache must not fail the run.
func (g *Gatherer) store(s inventory.Server, f Facts) {
	if g.CacheDir == "" || g.TTL <= 0 {
		return
	}
	path := g.path(s)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	e := entry{Address: s.Address, Collected: g.now(), Facts: f}
	_ = output.WriteFileAtomic(path, 0o600, func(w io.Writer) error {
		return json.NewEncoder(w).Encode(e)
	})
}
