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
			inv, err := tt.task.Render()
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
