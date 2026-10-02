package hostd

import (
	"errors"
	"os/exec"
	"reflect"
	"testing"
)

func withFakeRunCommand(t *testing.T, fake func(name string, args ...string) error) {
	t.Helper()
	original := runCommand
	runCommand = fake
	t.Cleanup(func() { runCommand = original })
}

func TestSuspendRunsTheDefaultCommand(t *testing.T) {
	var gotName string
	var gotArgs []string
	withFakeRunCommand(t, func(name string, args ...string) error {
		gotName, gotArgs = name, args
		return nil
	})

	if err := NewSystemSuspender().Suspend(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotName != DefaultSuspendCommand[0] || !reflect.DeepEqual(gotArgs, DefaultSuspendCommand[1:]) {
		t.Fatalf("got command %q %v, want %q %v", gotName, gotArgs, DefaultSuspendCommand[0], DefaultSuspendCommand[1:])
	}
}

func TestDefaultCommandIsSystemctlSuspendOnly(t *testing.T) {
	// Suspend-to-RAM only -- never a full shutdown (ACPI S5).
	want := []string{"systemctl", "suspend"}
	if !reflect.DeepEqual(DefaultSuspendCommand, want) {
		t.Fatalf("got %v, want %v", DefaultSuspendCommand, want)
	}
}

func TestSuspendRunsAConfiguredCommand(t *testing.T) {
	var gotName string
	var gotArgs []string
	withFakeRunCommand(t, func(name string, args ...string) error {
		gotName, gotArgs = name, args
		return nil
	})

	suspender, err := NewSystemSuspenderWithCommand([]string{"some-other-suspend-tool", "--now"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := suspender.Suspend(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotName != "some-other-suspend-tool" || !reflect.DeepEqual(gotArgs, []string{"--now"}) {
		t.Fatalf("got command %q %v, want %q %v", gotName, gotArgs, "some-other-suspend-tool", []string{"--now"})
	}
}

func TestSuspendPropagatesErrorFromExitCode(t *testing.T) {
	wantErr := &exec.ExitError{}
	withFakeRunCommand(t, func(name string, args ...string) error {
		return wantErr
	})

	err := NewSystemSuspender().Suspend()
	if !errors.Is(err, error(wantErr)) {
		t.Fatalf("got err %v, want %v", err, wantErr)
	}
}

func TestSuspendPropagatesErrorWhenCommandIsNotFound(t *testing.T) {
	wantErr := errors.New("no such file")
	withFakeRunCommand(t, func(name string, args ...string) error {
		return wantErr
	})

	err := NewSystemSuspender().Suspend()
	if !errors.Is(err, wantErr) {
		t.Fatalf("got err %v, want %v", err, wantErr)
	}
}

func TestConstructorRejectsAnEmptyCommand(t *testing.T) {
	// Fails fast at startup rather than mid-request.
	if _, err := NewSystemSuspenderWithCommand(nil); err == nil {
		t.Fatal("NewSystemSuspenderWithCommand(nil) succeeded, want error")
	}
	if _, err := NewSystemSuspenderWithCommand([]string{}); err == nil {
		t.Fatal("NewSystemSuspenderWithCommand([]string{}) succeeded, want error")
	}
}
