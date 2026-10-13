package cli

import (
	"os"
	"strings"
	"testing"
)

func TestColorModeEnv(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		flag string
		want bool
	}{
		{name: "no_color wins", env: map[string]string{"NO_COLOR": ""}, flag: "auto", want: false},
		{name: "dumb terminal", env: map[string]string{"TERM": "dumb"}, flag: "auto", want: false},
		{name: "flag always overrides no_color", env: map[string]string{"NO_COLOR": "1"}, flag: "always", want: true},
		{name: "flag never", env: map[string]string{"CLICOLOR_FORCE": "1"}, flag: "never", want: false},
		{name: "clicolor force on pipe", env: map[string]string{"CLICOLOR_FORCE": "1"}, flag: "auto", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, k := range []string{"NO_COLOR", "TERM", "CLICOLOR_FORCE"} {
				_ = os.Unsetenv(k)
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			opts := &Options{Color: tt.flag, Out: &strings.Builder{}}
			if got := colorMode(opts); got != tt.want {
				t.Fatalf("colorMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOSC8(t *testing.T) {
	got := osc8("https://github.com/o/r/issues/1", "o/r#1", true)
	if !strings.Contains(got, "\x1b]8;;https://github.com/o/r/issues/1") || !strings.Contains(got, "o/r#1") {
		t.Fatalf("osc8 enabled = %q", got)
	}
	if got := osc8("https://x", "plain", false); got != "plain" {
		t.Fatalf("osc8 disabled = %q, want plain", got)
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("hello world", 8); got != "hello w…" {
		t.Fatalf("truncate = %q", got)
	}
	if got := truncateRunes("short", 10); got != "short" {
		t.Fatalf("truncate short = %q", got)
	}
}
