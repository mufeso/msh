package errs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"example.com/msh/internal/exitcode"
)

func TestErrorFormatAndUnwrap(t *testing.T) {
	err := New(KindAuth, "web01", io.EOF)
	if got := err.Error(); got != "web01: authentication failed: EOF" {
		t.Errorf("Error() = %q", got)
	}
	if !errors.Is(err, io.EOF) {
		t.Error("errors.Is cannot see the cause")
	}
}

func TestKindOfThroughWrapping(t *testing.T) {
	wrapped := fmt.Errorf("context: %w", New(KindHostKey, "db01", nil))
	if KindOf(wrapped) != KindHostKey {
		t.Errorf("KindOf = %v", KindOf(wrapped))
	}
	if KindOf(context.Canceled) != KindCanceled {
		t.Error("context.Canceled not recognized")
	}
	if KindOf(errors.New("x")) != KindUnknown {
		t.Error("plain error should be unknown")
	}
}

func TestExitCodes(t *testing.T) {
	tests := map[Kind]exitcode.Code{
		KindDNS: exitcode.Connection, KindAuth: exitcode.Connection, KindHostKey: exitcode.Connection,
		KindConnectTimeout: exitcode.Timeout, KindCanceled: exitcode.Canceled, KindTask: exitcode.Config,
		KindScriptTransfer: exitcode.Transfer, KindTmuxUnavailable: exitcode.Tmux, KindUnknown: exitcode.Internal,
	}
	for k, want := range tests {
		if got := ExitCode(k); got != want {
			t.Errorf("ExitCode(%v) = %v, want %v", k, got, want)
		}
	}
}

func TestEveryKindHasText(t *testing.T) {
	for k := KindUnknown; k <= KindInternal; k++ {
		if _, ok := kindText[k]; !ok {
			t.Errorf("kind %d has no text", k)
		}
	}
}

func TestRole(t *testing.T) {
	err := NewRole(KindAuth, RoleBastion, "jump1", io.EOF)
	if got := err.Error(); got != "bastion jump1: authentication failed: EOF" {
		t.Errorf("Error() = %q", got)
	}
	if RoleOf(fmt.Errorf("x: %w", err)) != RoleBastion {
		t.Error("RoleOf through wrapping failed")
	}
	if RoleOf(New(KindTCP, "web01", nil)) != RoleTarget {
		t.Error("default role should be target")
	}
}

func TestVia(t *testing.T) {
	e := New(KindAuth, "app01", io.EOF)
	e.Via = "jump"
	if got := e.Error(); got != "app01 (via bastion jump): authentication failed: EOF" {
		t.Errorf("Error() = %q", got)
	}
}
