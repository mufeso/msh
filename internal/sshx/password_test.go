package sshx

import (
	"reflect"
	"testing"
)

func TestAnswerChallenge(t *testing.T) {
	ask := func() (string, error) { return "value", nil }
	got, err := answerChallenge([]string{"Password: "}, []bool{false}, ask)
	if err != nil || !reflect.DeepEqual(got, []string{"value"}) {
		t.Errorf("got %q, %v", got, err)
	}
	if got, err := answerChallenge(nil, nil, ask); err != nil || len(got) != 0 {
		t.Errorf("an empty challenge must succeed: %q, %v", got, err)
	}
	for _, q := range []string{"Verification code: ", "One-time token: "} {
		if _, err := answerChallenge([]string{q}, []bool{false}, ask); err == nil {
			t.Errorf("%q received the password", q)
		}
	}
	if _, err := answerChallenge([]string{"Password: "}, []bool{true}, ask); err == nil {
		t.Error("a question that echoes received the password")
	}
}
