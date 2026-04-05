package output

import (
	"io"
	"os"
	"path/filepath"
)

// WriteFileAtomic writes a file so that readers see either the old
// content or the complete new content, never a partial file:
//
//  1. write to a temporary file in the same directory,
//  2. flush it to disk (Sync),
//  3. rename it over the destination (atomic within a file system).
//
// On any error the temporary file is removed.
func WriteFileAtomic(path string, perm os.FileMode, write func(io.Writer) error) (err error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".msh-tmp-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	if err = tmp.Chmod(perm); err != nil {
		return err
	}
	if err = write(tmp); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// NewRunView builds the JSON document of a run.
func NewRunView(results []ResultView, summary SummaryView) RunView {
	if results == nil {
		results = []ResultView{}
	}
	return RunView{Results: results, Summary: summary}
}
