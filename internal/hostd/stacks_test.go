package hostd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type eventRecorder struct{ events []string }

func (r *eventRecorder) SendEvent(eventType, outcome string, identity *string, extra map[string]any) {
	if s, ok := extra["stack"]; ok {
		eventType += ":" + s.(string)
	}
	r.events = append(r.events, eventType)
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

// push commits files (path -> content, "" deletes) to origin's main.
func push(t *testing.T, work string, files map[string]string) {
	t.Helper()
	for p, content := range files {
		full := filepath.Join(work, p)
		if content == "" {
			os.Remove(full)
			continue
		}
		os.MkdirAll(filepath.Dir(full), 0o755)
		os.WriteFile(full, []byte(content), 0o644)
	}
	git(t, work, "add", "-A")
	git(t, work, "commit", "-q", "-m", "change")
	git(t, work, "push", "-q", "origin", "HEAD:main")
}

func TestStackSync(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("needs git")
	}
	root := t.TempDir()
	origin, work, clone := filepath.Join(root, "origin"), filepath.Join(root, "work"), filepath.Join(root, "clone")
	git(t, root, "init", "-q", "--bare", "-b", "main", origin)
	git(t, root, "clone", "-q", origin, work)
	push(t, work, map[string]string{"a/compose.yml": "a1", "b/compose.yml": "b1", "README.md": "x"})
	git(t, root, "clone", "-q", origin, clone)

	rec := &eventRecorder{}
	s := NewStackSync(clone, "", rec)
	realRun := s.run
	var calls []string
	s.run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "git" {
			return realRun(ctx, name, args...)
		}
		calls = append(calls, name+" "+strings.Join(args, " "))
		if name == "sops" {
			if strings.Contains(args[len(args)-1], "broken") {
				return nil, errors.New("no key")
			}
			return []byte("SECRET=1\n"), nil
		}
		return nil, nil
	}
	sync := func() {
		t.Helper()
		calls, rec.events = nil, nil
		if err := s.Sync(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	sync()
	if len(calls) != 0 || strings.Join(rec.events, ",") != "stacks_adopted" {
		t.Fatalf("first sync should adopt, not deploy: calls=%v events=%v", calls, rec.events)
	}

	sync()
	if len(calls) != 0 || len(rec.events) != 0 {
		t.Fatalf("nothing changed: calls=%v events=%v", calls, rec.events)
	}

	push(t, work, map[string]string{"a/compose.yml": "a2", "a/secrets.env": "enc", "README.md": "y", "broken/compose.yml": "c", "broken/secrets.env": "enc"})
	sync()
	if got := strings.Join(rec.events, ","); got != "stack_deployed:a,stack_skipped:broken" {
		t.Fatalf("events=%v calls=%v", got, calls)
	}
	if env, _ := os.ReadFile(filepath.Join(clone, "a", ".env")); string(env) != "SECRET=1\n" {
		t.Fatalf("a/.env = %q", env)
	}
	if !strings.Contains(strings.Join(calls, "\n"), "docker compose -f a/compose.yml up -d --remove-orphans") {
		t.Fatalf("calls=%v", calls)
	}

	push(t, work, map[string]string{"a/compose.yml": "", "a/secrets.env": ""})
	sync()
	if got := strings.Join(rec.events, ","); got != "stack_removed:a" {
		t.Fatalf("events=%v calls=%v", got, calls)
	}
	if _, err := os.Stat(filepath.Join(clone, "a", ".env")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a removed stack's decrypted .env should be deleted")
	}
}
