package cli

import "testing"

// (A11) Account resolution: flag beats BOARDWALK_ACCOUNT; empty
// means gh's active account.
func TestAccountNameResolution(t *testing.T) {
	t.Setenv("BOARDWALK_ACCOUNT", "env-user")
	if got := (&Options{Account: "flag-user"}).AccountName(); got != "flag-user" {
		t.Fatalf("flag should win, got %q", got)
	}
	if got := (&Options{}).AccountName(); got != "env-user" {
		t.Fatalf("env should apply, got %q", got)
	}
	t.Setenv("BOARDWALK_ACCOUNT", "")
	if got := (&Options{}).AccountName(); got != "" {
		t.Fatalf("empty should stay empty, got %q", got)
	}
}
