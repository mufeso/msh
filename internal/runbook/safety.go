package runbook

import "example.com/msh/internal/task"

// rank orders safety classes from harmless to dangerous.
func rank(s task.Safety) int {
	switch s {
	case task.SafetyDangerous:
		return 2
	case task.SafetyChanges:
		return 1
	default:
		return 0
	}
}

// Safety returns the class of the most dangerous step. A command step
// counts as "changes" unless it declares read_only: msh cannot know what
// an arbitrary command does, so it assumes the unsafe answer.
func (rb *Runbook) Safety(tasks *task.Set) task.Safety {
	worst := task.SafetyReadOnly
	for _, s := range rb.Steps {
		c := task.SafetyChanges
		if s.Command != "" && s.ReadOnly {
			c = task.SafetyReadOnly
		}
		if s.Task != "" {
			if t, ok := tasks.Get(s.Task); ok {
				c = t.EffectiveSafety()
			}
		}
		if rank(c) > rank(worst) {
			worst = c
		}
	}
	return worst
}
