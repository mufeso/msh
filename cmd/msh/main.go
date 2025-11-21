package main

import (
	"os"

	"example.com/msh/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr))
}
