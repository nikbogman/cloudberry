// Suspend-to-RAM: runs the configured system command as a subprocess. This
// is the one external side effect in this app — SystemSuspender.Suspend is
// the seam mocked in handler tests.
//
// No code path here, or anywhere else in this app, can trigger a full
// shutdown (ACPI S5).
package compute

import (
	"errors"
	"os/exec"
)

// DefaultSuspendCommand suspends to RAM only — never a full shutdown.
var DefaultSuspendCommand = []string{"systemctl", "suspend"}

// runCommand is a seam over exec.Command(...).Run(), swapped out in tests
// (the Go analogue of monkeypatching subprocess.run).
var runCommand = func(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

// SystemSuspender suspends the host to RAM by running the configured command.
type SystemSuspender struct {
	command []string
}

// NewSystemSuspender returns a SystemSuspender running DefaultSuspendCommand
// — the only constructor real callers should use.
func NewSystemSuspender() *SystemSuspender {
	return &SystemSuspender{command: DefaultSuspendCommand}
}

// NewSystemSuspenderWithCommand returns a SystemSuspender running command
// instead of the default. Only intended for tests: it errors on an empty
// command, failing fast at construction rather than mid-request.
func NewSystemSuspenderWithCommand(command []string) (*SystemSuspender, error) {
	if len(command) == 0 {
		return nil, errors.New("suspend command must not be empty")
	}
	return &SystemSuspender{command: command}, nil
}

// Suspend runs the configured suspend command, propagating any error
// (a non-zero exit or a not-found binary) as-is.
func (s *SystemSuspender) Suspend() error {
	return runCommand(s.command[0], s.command[1:]...)
}
