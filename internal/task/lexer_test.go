package task

import "testing"

func TestCheckShellTemplate(t *testing.T) {
	tests := []struct {
		tmpl    string
		redir   bool
		wantErr bool
	}{
		{"df -h | sort -k5", false, false},
		{"grep -- {{p}} /var/log/messages | tail -n 20", false, false},
		{"printf '%s\\n' {{p}}", false, false},
		{"echo 'a > b' \"c < d\"", false, false},
		{"echo \\> literal", false, false},
		{"set -e\nuptime\nfree -m", false, false},
		{"grep '{{p}}' file", false, true},
		{"grep \"{{p}}\" file", false, true},
		{"echo x > /etc/motd", false, true},
		{"echo x > /tmp/out", true, false},
		{"cat < /etc/hostname", false, true},
		{"echo 'unterminated", false, true},
	}
	for _, tt := range tests {
		err := checkShellTemplate(tt.tmpl, tt.redir)
		if (err != nil) != tt.wantErr {
			t.Errorf("checkShellTemplate(%q, %v) = %v, wantErr %v", tt.tmpl, tt.redir, err, tt.wantErr)
		}
	}
}

func TestPlaceholdersNeedPOSIXShell(t *testing.T) {
	_, err := NewSet(map[string]Task{"py": {
		Interpreter: "python3",
		Command:     "print({{x}})",
		Arguments:   []Argument{{Name: "x"}},
	}})
	if err == nil {
		t.Error("placeholder in a python -c command accepted")
	}
	if _, err := NewSet(map[string]Task{"py": {Interpreter: "python3", Command: "print(1)"}}); err != nil {
		t.Errorf("python command without placeholders rejected: %v", err)
	}
}
