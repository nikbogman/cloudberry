package hostd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/nikbogman/cloudberry/internal/routes"
	"github.com/nikbogman/cloudberry/internal/tailnet"
)

// DefaultIdleMinutes is the routes file's default idle time, as in cmd/routes.
const DefaultIdleMinutes = 30

// StackSync applies the deploys repo, cloned at Dir, to Docker: each stack
// whose folder changed since the last sync is brought up (or down, if its
// folder is gone), after decrypting its secrets.env to .env with sops. The
// last applied commit is the `deployed` branch in the clone, so nothing else
// stores state. It never reads Docker's state to decide anything: the repo is
// the source of truth.
type StackSync struct {
	Dir    string
	Logger EventLogger
	// FirewallDropIn overrides stack-firewall's PORTS; empty skips the firewall.
	FirewallDropIn string

	run     func(ctx context.Context, name string, args ...string) ([]byte, error)
	mu      sync.Mutex
	started bool // the first sync since start decrypts and checks the firewall even if nothing changed
}

func NewStackSync(dir, firewallDropIn string, logger EventLogger) *StackSync {
	s := &StackSync{Dir: dir, FirewallDropIn: firewallDropIn, Logger: logger}
	s.run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return out, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, tail(out))
		}
		return out, nil
	}
	return s
}

// Sync fetches main and applies what changed. The first sync on a clone
// with no `deployed` branch adopts main as already deployed, so starting the
// sync never restarts a running stack or starts one that was left down.
func (s *StackSync) Sync(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.run(ctx, "git", "fetch", "--quiet", "origin", "main"); err != nil {
		return err
	}
	target, err := s.rev(ctx, "origin/main")
	if err != nil {
		return err
	}
	deployed, _ := s.rev(ctx, "refs/heads/deployed")
	if deployed == target && s.started {
		return nil
	}
	if _, err := s.run(ctx, "git", "reset", "--hard", "--quiet", target); err != nil {
		return err
	}
	undecrypted := s.decryptAll(ctx)

	if deployed == "" {
		s.Logger.SendEvent("stacks_adopted", "succeeded", nil, map[string]any{"commit": target})
	} else if deployed != target {
		changed, err := s.changedStacks(ctx, deployed, target)
		if err != nil {
			return err
		}
		for _, name := range changed {
			s.apply(ctx, name, undecrypted[name], target)
		}
	}
	// Recorded even if a stack failed: it's retried on the next push, not
	// every few minutes, and its stack_failed event says why.
	if _, err := s.run(ctx, "git", "update-ref", "refs/heads/deployed", target); err != nil {
		return err
	}
	s.started = true
	return s.updateFirewall(ctx)
}

func (s *StackSync) rev(ctx context.Context, ref string) (string, error) {
	out, err := s.run(ctx, "git", "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	return strings.TrimSpace(string(out)), err
}

// changedStacks lists the top-level folders touched between two commits that
// hold a compose.yml at either end; root files and other folders aren't stacks.
func (s *StackSync) changedStacks(ctx context.Context, from, to string) ([]string, error) {
	out, err := s.run(ctx, "git", "diff", "--name-only", from, to)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var names []string
	for _, f := range strings.Fields(string(out)) {
		name, _, ok := strings.Cut(f, "/")
		if !ok || seen[name] {
			continue
		}
		seen[name] = true
		_, errNow := s.run(ctx, "git", "cat-file", "-e", to+":"+name+"/compose.yml")
		_, errBefore := s.run(ctx, "git", "cat-file", "-e", from+":"+name+"/compose.yml")
		if errNow == nil || errBefore == nil {
			names = append(names, name)
		}
	}
	sort.Strings(names) // db sorts before the stacks that use it
	return names, nil
}

// apply brings one stack up, or down if its folder is gone. A stack whose
// secrets didn't decrypt is skipped, never started with a stale or missing
// .env. Compose project names are the folder names.
func (s *StackSync) apply(ctx context.Context, name string, decryptErr error, commit string) {
	extra := map[string]any{"stack": name, "commit": commit}
	compose := filepath.Join(name, "compose.yml")
	if _, err := os.Stat(filepath.Join(s.Dir, compose)); errors.Is(err, os.ErrNotExist) {
		if _, err := s.run(ctx, "docker", "compose", "-p", name, "down", "--remove-orphans"); err != nil {
			extra["error"] = err.Error()
			s.Logger.SendEvent("stack_failed", "failed", nil, extra)
			return
		}
		os.Remove(filepath.Join(s.Dir, name, ".env")) // decrypted secrets of a stack that's gone
		s.Logger.SendEvent("stack_removed", "succeeded", nil, extra)
		return
	}
	if decryptErr != nil {
		extra["error"] = decryptErr.Error()
		s.Logger.SendEvent("stack_skipped", "failed", nil, extra)
		return
	}
	if _, err := s.run(ctx, "docker", "compose", "-f", compose, "up", "-d", "--remove-orphans"); err != nil {
		extra["error"] = err.Error()
		s.Logger.SendEvent("stack_failed", "failed", nil, extra)
		return
	}
	s.Logger.SendEvent("stack_deployed", "succeeded", nil, extra)
}

// decryptAll writes every <stack>/secrets.env, decrypted, to <stack>/.env
// (mode 600, gitignored), only where it changed. sops finds blackberry's age
// key through SOPS_AGE_KEY_FILE, from hostd's unit. Returns the stacks that
// failed.
func (s *StackSync) decryptAll(ctx context.Context) map[string]error {
	failed := map[string]error{}
	files, _ := filepath.Glob(filepath.Join(s.Dir, "*", "secrets.env"))
	for _, f := range files {
		name := filepath.Base(filepath.Dir(f))
		rel := filepath.Join(name, "secrets.env")
		plain, err := s.run(ctx, "sops", "decrypt", "--input-type", "dotenv", "--output-type", "dotenv", rel)
		if err == nil {
			err = writeIfChanged(filepath.Join(s.Dir, name, ".env"), plain, 0o600)
		}
		if err != nil {
			failed[name] = err
		}
	}
	return failed
}

// updateFirewall points stack-firewall's PORTS at the routed stacks' ports,
// through a drop-in that overrides the unit's own value.
func (s *StackSync) updateFirewall(ctx context.Context) error {
	if s.FirewallDropIn == "" {
		return nil
	}
	cfg, _, err := routes.FromDir(s.Dir, DefaultIdleMinutes)
	if err != nil {
		return err
	}
	ports := strings.Join(routes.Ports(cfg), " ")
	want := []byte("# Written by hostd from the deploys repo's x-route blocks.\n[Service]\nEnvironment=\"PORTS=" + ports + "\"\n")
	if cur, _ := os.ReadFile(s.FirewallDropIn); bytes.Equal(cur, want) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.FirewallDropIn), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(s.FirewallDropIn, want, 0o644); err != nil {
		return err
	}
	if _, err := s.run(ctx, "systemctl", "daemon-reload"); err != nil {
		return err
	}
	// restart runs ExecStop with the old PORTS, then ExecStart with the new.
	if _, err := s.run(ctx, "systemctl", "restart", "stack-firewall"); err != nil {
		return err
	}
	s.Logger.SendEvent("firewall_updated", "succeeded", nil, map[string]any{"ports": ports})
	return nil
}

// RoutesHandler serves gateway's routes file, built from the synced stacks.
// gateway polls it, so a route change in deploys reaches raspberry without
// GitHub credentials on the Pi.
func (s *StackSync) RoutesHandler() http.Handler {
	return tailnet.RequireTailnetIdentity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		_, data, err := routes.FromDir(s.Dir, DefaultIdleMinutes)
		s.mu.Unlock()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
}

func writeIfChanged(path string, data []byte, perm os.FileMode) error {
	if cur, err := os.ReadFile(path); err == nil && bytes.Equal(cur, data) {
		return nil
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// tail keeps the end of a command's output, where the error usually is, so an
// event stays small.
func tail(out []byte) string {
	s := strings.TrimSpace(string(out))
	if len(s) > 500 {
		s = "…" + s[len(s)-500:]
	}
	return s
}
