package cli

import (
	"fmt"
	"os"
	"os/exec"
)

// Editor is the $EDITOR external-process boundary `packets amend` needs.
// Tests use a hand-rolled stub that edits the file directly instead of
// launching a real editor (CONVENTIONS: no test may invoke a real editor).
type Editor interface {
	Open(path string) error
}

// ExecEditor is the real Editor: it execs $EDITOR path, wired to the
// calling process's own stdio so a human's terminal editor works.
type ExecEditor struct{}

func (ExecEditor) Open(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return fmt.Errorf("amend: $EDITOR is not set")
	}
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("amend: %s %s: %s", editor, path, err)
	}
	return nil
}
