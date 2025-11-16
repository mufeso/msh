// Command msh is "My Parallel SSH", a command-line SSH fleet-management
// tool for Linux servers.
//
// In step001 it only prints its name. Later steps add the CLI, inventory,
// SSH execution, concurrency, tasks, and interactive sessions.
package main

import "fmt"

// Application identity. These are constants because they never change
// while the program runs.
const (
	appName     = "msh"
	appLongName = "My Parallel SSH"
)

// banner returns the application's display name.
//
// main only prints; banner computes the text. Keeping them separate lets a
// test check the text without capturing the program's output.
func banner() string {
	return appName + " — " + appLongName
}

func main() {
	fmt.Println(banner())
}
