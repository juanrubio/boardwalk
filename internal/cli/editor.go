package cli

import "os/exec"

func newEditorCmd(name string, args []string) *exec.Cmd {
	return exec.Command(name, args...) //nolint:gosec // user's configured editor
}
