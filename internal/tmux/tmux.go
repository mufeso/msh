// Package tmux opens several interactive msh sessions side by side in
// tmux panes. It only touches tmux sessions it created itself.
package tmux

import (
	"errors"

	"example.com/msh/internal/errs"
)

// LookPathFunc has the signature of exec.LookPath.
type LookPathFunc func(file string) (string, error)

// Find returns the path of the tmux executable. msh never installs
// software: a missing tmux is reported as KindTmuxUnavailable.
func Find(lookPath LookPathFunc) (string, error) {
	path, err := lookPath("tmux")
	if err != nil {
		return "", errs.New(errs.KindTmuxUnavailable, "",
			errors.New("tmux was not found in PATH; install it with your package manager (for example: dnf install tmux, apt install tmux)"))
	}
	return path, nil
}
