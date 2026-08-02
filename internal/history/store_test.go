package history

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

func saved(t *testing.T, s Store, command string, at time.Time) Record {
	t.Helper()
	r := NewRecord(command, sampleReport("a"), at)
	if err := s.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestAllNewestFirst(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, c := range []string{"one", "two", "three"} {
		saved(t, s, c, base.Add(time.Duration(i)*time.Hour))
	}
	var got []string
	for r, err := range s.All() {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, r.Command)
	}
	if !slices.Equal(got, []string{"three", "two", "one"}) {
		t.Errorf("order = %v", got)
	}
	n := 0
	for range s.All() {
		n++
		break // stopping early must work
	}
	if n != 1 {
		t.Errorf("n = %d", n)
	}
}

func TestFind(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	a := saved(t, s, "first", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	b := saved(t, s, "second", time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))
	if r, err := s.Find("last"); err != nil || r.ID != b.ID {
		t.Errorf("last = %+v, %v", r, err)
	}
	if r, err := s.Find(a.ID[:8]); err != nil || r.ID != a.ID {
		t.Errorf("prefix = %+v, %v", r, err)
	}
	if _, err := s.Find("202"); err == nil {
		t.Error("an ambiguous prefix was accepted")
	}
	if _, err := s.Find("1999"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v", err)
	}
	if _, err := (Store{Dir: t.TempDir()}).Find("last"); !errors.Is(err, ErrNotFound) {
		t.Errorf("empty store: err = %v", err)
	}
}

func TestPrune(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range 5 {
		saved(t, s, fmt.Sprint(i), base.Add(time.Duration(i)*time.Hour))
	}
	n, err := s.Prune(2)
	if err != nil || n != 3 {
		t.Fatalf("n = %d, err = %v", n, err)
	}
	var got []string
	for r, _ := range s.All() {
		got = append(got, r.Command)
	}
	if !slices.Equal(got, []string{"4", "3"}) {
		t.Errorf("kept = %v, want the newest", got)
	}
	if n, _ := s.Prune(10); n != 0 {
		t.Errorf("n = %d", n)
	}
}
