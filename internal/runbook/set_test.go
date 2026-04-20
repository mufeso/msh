package runbook

import (
	"strings"
	"testing"

	"example.com/msh/internal/task"
)

func testTasks(t *testing.T) *task.Set {
	t.Helper()
	s, err := task.NewSet(map[string]task.Task{
		"uptime":  {Argv: []string{"uptime"}},
		"restart": {Argv: []string{"systemctl", "restart", "{{service}}"}, Arguments: []task.Argument{{Name: "service", Required: true}}, Safety: task.SafetyChanges},
		"follow":  {Command: "tail -f /var/log/messages", Interactive: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNewSetValid(t *testing.T) {
	s, err := NewSet(map[string]Runbook{
		"b": {Steps: []Step{{Name: "one", Task: "uptime"}}},
		"a": {Steps: []Step{{Name: "one", Command: "df -h | sort"}, {Name: "two", Task: "uptime"}}},
	}, testTasks(t))
	if err != nil {
		t.Fatal(err)
	}
	if list := s.List(); len(list) != 2 || list[0].Name != "a" {
		t.Errorf("List() = %v", list)
	}
	if _, ok := s.Get("b"); !ok {
		t.Error("Get(b) not found")
	}
}

func TestNewSetProblems(t *testing.T) {
	bad := map[string]Runbook{
		"empty":       {},
		"no-action":   {Steps: []Step{{Name: "x"}}},
		"both":        {Steps: []Step{{Name: "x", Task: "uptime", Command: "ls"}}},
		"no-name":     {Steps: []Step{{Task: "uptime"}}},
		"dup":         {Steps: []Step{{Name: "x", Task: "uptime"}, {Name: "x", Task: "uptime"}}},
		"ghost":       {Steps: []Step{{Name: "x", Task: "nope"}}},
		"interactive": {Steps: []Step{{Name: "x", Task: "follow"}}},
		"retries":     {Steps: []Step{{Name: "x", Task: "uptime", Retries: 1000}}},
		"redirect":    {Steps: []Step{{Name: "x", Command: "echo x > /etc/motd"}}},
		"params":      {Steps: []Step{{Name: "x", Command: "ls", Params: map[string]string{"a": "b"}}}},
	}
	_, err := NewSet(bad, testTasks(t))
	if err == nil {
		t.Fatal("invalid runbooks accepted")
	}
	for name := range bad {
		if !strings.Contains(err.Error(), `runbook "`+name+`"`) {
			t.Errorf("no problem reported for %s:\n%v", name, err)
		}
	}
}
