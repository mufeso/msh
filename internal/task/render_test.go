package task

import "testing"

func TestRenderModes(t *testing.T) {
	tests := []struct {
		name      string
		task      Task
		wantCmd   string
		wantStdin string
	}{
		{"direct", Task{Argv: []string{"df", "-h", "/var/log"}}, "df -h /var/log", ""},
		{"direct quoted", Task{Argv: []string{"echo", "two words"}}, "echo 'two words'", ""},
		{"shell default sh", Task{Command: "df -h | sort"}, "sh -c 'df -h | sort'", ""},
		{"shell bash", Task{Interpreter: "bash", Command: "echo $HOME"}, "bash -c 'echo $HOME'", ""},
		{"script bash", Task{Interpreter: "bash", Script: "echo hi\n"}, "bash -s --", "echo hi\n"},
		{"script python", Task{Interpreter: "python3", Script: "print(1)\n"}, "python3 -", "print(1)\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv, err := tt.task.Render(nil)
			if err != nil {
				t.Fatal(err)
			}
			if inv.Command != tt.wantCmd || string(inv.Stdin) != tt.wantStdin {
				t.Errorf("got %q / %q, want %q / %q", inv.Command, inv.Stdin, tt.wantCmd, tt.wantStdin)
			}
		})
	}
}

func TestUnknownInterpreter(t *testing.T) {
	if _, err := NewSet(map[string]Task{"x": {Interpreter: "ruby", Command: "puts 1"}}); err == nil {
		t.Error("unknown interpreter accepted")
	}
}

func messageTask() Task {
	return Task{
		Name:        "message",
		Interpreter: "bash",
		Command:     "tail -100f /var/log/messages | grep --line-buffered -- {{pattern}}",
		Arguments:   []Argument{{Name: "pattern", Required: true}},
	}
}

func TestRenderPositionalArguments(t *testing.T) {
	tk := messageTask()
	values, err := tk.Bind([]string{"sshd"})
	if err != nil {
		t.Fatal(err)
	}
	inv, err := tk.Render(values)
	if err != nil {
		t.Fatal(err)
	}
	want := "bash -c 'tail -100f /var/log/messages | grep --line-buffered -- sshd'"
	if inv.Command != want {
		t.Errorf("Command = %s\nwant      %s", inv.Command, want)
	}

	values, _ = tk.Bind([]string{"Failed password"})
	inv, _ = tk.Render(values)
	want = `bash -c 'tail -100f /var/log/messages | grep --line-buffered -- '\''Failed password'\'''`
	if inv.Command != want {
		t.Errorf("Command = %s\nwant      %s", inv.Command, want)
	}
}

func TestRenderNumericPlaceholderAndArgv(t *testing.T) {
	tk := Task{Argv: []string{"journalctl", "-u", "{{1}}", "--since={{2}}"},
		Arguments: []Argument{{Name: "unit"}, {Name: "since"}}}
	values, _ := tk.Bind([]string{"sshd", "1 hour ago"})
	inv, err := tk.Render(values)
	if err != nil {
		t.Fatal(err)
	}
	if want := "journalctl -u sshd '--since=1 hour ago'"; inv.Command != want {
		t.Errorf("Command = %s, want %s", inv.Command, want)
	}
}

func TestRenderScriptArguments(t *testing.T) {
	tk := Task{Interpreter: "bash", Script: "echo \"$1\"\n", Arguments: []Argument{{Name: "who"}}}
	values, _ := tk.Bind([]string{"a b"})
	inv, _ := tk.Render(values)
	if inv.Command != "bash -s -- 'a b'" || string(inv.Stdin) != "echo \"$1\"\n" {
		t.Errorf("inv = %+v", inv)
	}
}

func TestPlaceholderValidation(t *testing.T) {
	bad := map[string]Task{
		"unknown":   {Command: "echo {{nope}}"},
		"too high":  {Command: "echo {{2}}", Arguments: []Argument{{Name: "a"}}},
		"in script": {Script: "echo {{a}}", Arguments: []Argument{{Name: "a"}}},
		"dup arg":   {Command: "echo", Arguments: []Argument{{Name: "a"}, {Name: "a"}}},
	}
	for name, tk := range bad {
		if _, err := NewSet(map[string]Task{"t": tk}); err == nil {
			t.Errorf("%s: invalid task accepted", name)
		}
	}
}

func TestBindTooMany(t *testing.T) {
	tk := messageTask()
	if _, err := tk.Bind([]string{"a", "b"}); err == nil {
		t.Error("too many arguments accepted")
	}
}
