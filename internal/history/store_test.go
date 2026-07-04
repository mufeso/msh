package history

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoad(t *testing.T) {
	s := Store{Dir: filepath.Join(t.TempDir(), "history")}
	r := NewRecord("msh exec --all", sampleReport("a"), time.Now())
	if err := s.Save(r); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Command != r.Command || got.Results[0].Stdout != "hello from a\n" {
		t.Errorf("record = %+v", got)
	}
	dir, _ := os.Stat(s.Dir)
	file, _ := os.Stat(s.path(r.ID))
	if dir.Mode().Perm() != 0o700 || file.Mode().Perm() != 0o600 {
		t.Errorf("modes = %v, %v", dir.Mode().Perm(), file.Mode().Perm())
	}
	if _, err := s.Load("nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v", err)
	}
}
