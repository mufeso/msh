package app

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"example.com/msh/internal/inventory"
)

// terminalApprover asks on the terminal before each next batch. One
// bufio.Reader serves all questions, so input typed ahead is not lost.
func (c *cli) terminalApprover() func(ctx context.Context, next, total int, batch []inventory.Server) (bool, error) {
	reader := bufio.NewReader(c.terminal().Stdin())
	return func(ctx context.Context, next, total int, batch []inventory.Server) (bool, error) {
		names := make([]string, 0, len(batch))
		for _, s := range batch {
			names = append(names, s.Name)
		}
		fmt.Fprintf(c.stderr, "Continue with batch %d/%d (%s)? [y/N] ", next, total, strings.Join(names, ", "))

		// A terminal read cannot be canceled, so it runs in a goroutine
		// and the select below can react to CTRL+C. The channel is
		// buffered, so the goroutine can always finish.
		answer := make(chan string, 1)
		go func() {
			line, _ := reader.ReadString('\n')
			answer <- strings.ToLower(strings.TrimSpace(line))
		}()
		select {
		case a := <-answer:
			return a == "y" || a == "yes", nil
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}
}
