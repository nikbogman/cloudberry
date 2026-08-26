package compute

import (
	"errors"
	"os/exec"
)

// DefaultSuspendCommand suspends to RAM only. No code path in this app can
// trigger a full shutdown (ACPI S5).
var DefaultSuspendCommand = []string{"systemctl", "suspend"}

var runCommand = func(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

type SystemSuspender struct {
	command []string
}

// NewSystemSuspender is the only constructor real callers should use.
func NewSystemSuspender() *SystemSuspender {
	return &SystemSuspender{command: DefaultSuspendCommand}
}

// NewSystemSuspenderWithCommand is for tests: it errors on an empty
// command, failing fast at construction rather than mid-request.
func NewSystemSuspenderWithCommand(command []string) (*SystemSuspender, error) {
	if len(command) == 0 {
		return nil, errors.New("suspend command must not be empty")
	}
	return &SystemSuspender{command: command}, nil
}

func (s *SystemSuspender) Suspend() error {
	return runCommand(s.command[0], s.command[1:]...)
}
