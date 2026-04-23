package runbook

import (
	"fmt"
	"time"

	"example.com/msh/internal/task"
)

// PlannedStep is a step resolved into what will actually run.
type PlannedStep struct {
	Step    Step
	Command string        // remote command line
	Stdin   []byte        // script text, for script tasks
	Timeout time.Duration // 0 = use the command_timeout setting
}

// Plan resolves every step into its remote command. Nothing is executed,
// so a plan can be printed (--dry-run) or tested as plain data.
func (rb *Runbook) Plan(tasks *task.Set, values task.Values) ([]PlannedStep, error) {
	plan := make([]PlannedStep, 0, len(rb.Steps))
	for _, s := range rb.Steps {
		inv, err := rb.renderStep(s, tasks, values)
		if err != nil {
			return nil, fmt.Errorf("step %q: %w", s.Name, err)
		}
		ps := PlannedStep{Step: s, Command: inv.Command, Stdin: inv.Stdin, Timeout: inv.Timeout}
		if s.Timeout != nil {
			ps.Timeout = time.Duration(*s.Timeout)
		}
		plan = append(plan, ps)
	}
	return plan, nil
}

// renderStep renders a task step or a command step.
func (rb *Runbook) renderStep(s Step, tasks *task.Set, values task.Values) (task.Invocation, error) {
	if s.Command != "" {
		t := rb.commandTask(s)
		return t.Render(values)
	}
	t, ok := tasks.Get(s.Task)
	if !ok {
		return task.Invocation{}, fmt.Errorf("unknown task %q", s.Task)
	}
	bound, err := t.Bind(s.Args, s.Params)
	if err != nil {
		return task.Invocation{}, err
	}
	return t.Render(bound)
}
