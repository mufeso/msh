// Command msh-docs writes the generated documentation. It runs through
// "go generate ./..."; the argument is the docs directory.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"example.com/msh/internal/app"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: msh-docs DOCS_DIR")
		os.Exit(2)
	}
	files := map[string]string{
		filepath.Join(os.Args[1], "man", "msh.1"): app.Manual(),
		filepath.Join(os.Args[1], "commands.md"):  app.Reference(),
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
