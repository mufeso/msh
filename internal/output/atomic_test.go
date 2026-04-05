package output

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := WriteFileAtomic(path, 0o600, func(w io.Writer) error {
		_, err := io.WriteString(w, "new")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	info, _ := os.Stat(path)
	if string(data) != "new" || info.Mode().Perm() != 0o600 {
		t.Errorf("content %q, mode %v", data, info.Mode().Perm())
	}
}

func TestWriteFileAtomicFailureKeepsOldFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")
	os.WriteFile(path, []byte("old"), 0o600)
	err := WriteFileAtomic(path, 0o600, func(w io.Writer) error {
		io.WriteString(w, "partial")
		return errors.New("disk full")
	})
	if err == nil {
		t.Fatal("error not returned")
	}
	if data, _ := os.ReadFile(path); string(data) != "old" {
		t.Errorf("old file changed to %q", data)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("temporary file left behind: %v", entries)
	}
}
