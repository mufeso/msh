package history

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSaveAndLoad(t *testing.T) {
	s := Store{Dir: filepath.Join(t.TempDir(), "history")}
	r := NewRecord([]string{"exec", "--all"}, "ops", sampleReport("a"), time.Now())
	if err := s.Save(r); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load(r.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Command != "msh exec --all" || got.Operator != "ops" || got.Results[0].Stdout != "hello from a\n" {
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
	r := NewRecord([]string{command}, "ops", sampleReport("a"), at)
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
		got = append(got, r.Args[0])
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
		got = append(got, r.Args[0])
	}
	if !slices.Equal(got, []string{"4", "3"}) {
		t.Errorf("kept = %v, want the newest", got)
	}
	if n, _ := s.Prune(10); n != 0 {
		t.Errorf("n = %d", n)
	}
}

func TestLockIsExclusive(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	var inside, overlaps atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.withLock(func() error {
				if inside.Add(1) > 1 {
					overlaps.Add(1)
				}
				time.Sleep(2 * time.Millisecond)
				inside.Add(-1)
				return nil
			})
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if overlaps.Load() != 0 {
		t.Errorf("%d overlapping critical sections", overlaps.Load())
	}
}

func TestConcurrentSavesAndPrunes(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Save(NewRecord([]string{fmt.Sprint(i)}, "ops", sampleReport("a"), time.Now())); err != nil {
				t.Error(err)
			}
			if _, err := s.Prune(5); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if _, err := s.Prune(5); err != nil {
		t.Fatal(err)
	}
	ids, _ := s.ids()
	if len(ids) != 5 {
		t.Errorf("%d records, want 5", len(ids))
	}
}

func copyFixture(t *testing.T, dir, id string) {
	t.Helper()
	data, err := os.ReadFile("testdata/schema1.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestReadsOldSchema(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	id := "20260101T000000.000000-abc123"
	copyFixture(t, s.Dir, id)
	r, err := s.Load(id)
	if err != nil {
		t.Fatal(err)
	}
	if r.Schema != Schema || r.Operator != "unknown" || !slices.Equal(r.Args, []string{"exec", "--all", "--command", "true"}) {
		t.Errorf("record = %+v", r)
	}
}

func TestRefusesNewerSchema(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	id := "20990101T000000.000000-abc123"
	if err := os.WriteFile(filepath.Join(s.Dir, id+".json"), []byte(`{"schema": 99}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(id); err == nil || !strings.Contains(err.Error(), "newer msh") {
		t.Errorf("err = %v", err)
	}
}

func TestFilterIsLazy(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range 6 {
		saved(t, s, fmt.Sprint(i), base.Add(time.Duration(i)*time.Hour))
	}
	calls := 0
	even := Filter(s.All(), func(r Record) bool {
		calls++
		return r.Args[0] == "5" || r.Args[0] == "3"
	})
	for r, err := range even {
		if err != nil {
			t.Fatal(err)
		}
		if r.Args[0] == "5" {
			break
		}
	}
	if calls != 1 {
		t.Errorf("the filter looked at %d records after an early break", calls)
	}
}
