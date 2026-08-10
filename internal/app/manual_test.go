package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManualCoversEverything(t *testing.T) {
	m := Manual()
	for _, want := range []string{".SS packages check", ".SS runbook run", ".SS history diff", ".SH EXIT STATUS", ".B 130"} {
		if !strings.Contains(m, want) {
			t.Errorf("manual lacks %q", want)
		}
	}
	for _, line := range strings.Split(m, "\n") {
		if strings.HasPrefix(line, ".") {
			continue
		}
		if strings.Contains(line, " -") && !strings.Contains(line, `\-`) {
			t.Errorf("unescaped dash: %q", line)
		}
	}
}

func TestGeneratedManualIsUpToDate(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "man", "msh.1"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != Manual() {
		t.Error("docs/man/msh.1 is out of date; run: go generate ./...")
	}
}
