package compute

import (
	"log"
	"sync"
	"time"
)

// IdleWatcher runs continuously inside Compute API, polling once a minute
// for whether Compute counts as idle: no workload container CPU/network
// activity above baseline, no request proxied to a workload within the
// current idle window, and no active Hold. After idleTimeout with none of
// those true, it triggers the exact same Suspender.Suspend() the manual
// /suspend handler calls -- automatic and manual suspend are two callers
// of one code path, not two implementations.
type IdleWatcher struct {
	suspender   Suspender
	runtime     ContainerRuntime
	logger      EventLogger
	holdActive  func() bool
	lastProxied func() time.Time
	now         func() time.Time
	idleTimeout time.Duration
	dryRun      bool

	mu           sync.Mutex
	lastActivity time.Time
}

// NewIdleWatcher resets last-activity to now -- the process-start reset.
// The other reset (immediately after Suspend returns) happens inside poll,
// since suspend-to-RAM freezes the process rather than restarting it, so
// process-start alone would never fire again on wake.
func NewIdleWatcher(suspender Suspender, runtime ContainerRuntime, logger EventLogger, holdActive func() bool, lastProxied func() time.Time, idleTimeout time.Duration, dryRun bool) *IdleWatcher {
	return &IdleWatcher{
		suspender:    suspender,
		runtime:      runtime,
		logger:       logger,
		holdActive:   holdActive,
		lastProxied:  lastProxied,
		now:          time.Now,
		idleTimeout:  idleTimeout,
		dryRun:       dryRun,
		lastActivity: time.Now(),
	}
}

// Run polls once a minute until the process ends; it never returns on its
// own, matching the way this app already runs forever behind
// log.Fatal(http.ListenAndServe(...)).
func (w *IdleWatcher) Run() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		w.poll()
	}
}

// poll evaluates the idle signal for the current tick and, once idle for
// idleTimeout, triggers suspend. Split out from Run so tests can drive one
// tick at a time against a fake clock instead of a real ticker.
func (w *IdleWatcher) poll() {
	now := w.now()

	containerActive, err := w.runtime.ActivityAboveBaseline()
	if err != nil {
		// Fail toward staying awake, not toward suspending on a Docker
		// hiccup: an unreadable signal is treated as activity.
		log.Printf("idle watcher: checking container activity: %v", err)
		containerActive = true
	}

	idleDuration, trigger := w.evaluate(now, containerActive, w.holdActive(), w.lastProxied())
	if !trigger {
		return
	}

	extra := map[string]any{"idleSeconds": idleDuration.Seconds()}

	if w.dryRun {
		// Same trigger, same decision logic -- only the final call differs.
		// Reset last-activity so the watcher keeps cycling instead of
		// re-triggering on every poll for the rest of the session.
		w.logger.SendEvent("suspend_auto_dry_run", "dry_run", nil, extra)
		w.setLastActivity(now)
		return
	}

	w.logger.SendEvent("suspend_auto_triggered", "triggered", nil, extra)
	if err := w.suspender.Suspend(); err != nil {
		w.logger.SendEvent("suspend_auto_failed", "failed", nil, extra)
		return
	}
	w.logger.SendEvent("suspend_auto_succeeded", "succeeded", nil, extra)
	// Suspend() only returns once the machine has actually resumed, so its
	// return is itself the exact, guaranteed wake signal.
	w.setLastActivity(w.now())
}

// evaluate is the idle watcher's decision core: given the current time and
// a snapshot of the three activity signals, it advances lastActivity and
// reports the current idle duration plus whether idleTimeout has been
// reached. Kept as pure a function of its inputs as the running
// last-activity clock allows, so it's testable against fake
// activity/hold/clock inputs with no real polling or sleeping involved.
func (w *IdleWatcher) evaluate(now time.Time, containerActive, holdActive bool, lastProxied time.Time) (idleDuration time.Duration, trigger bool) {
	lastActivity := w.getLastActivity()
	if lastProxied.After(lastActivity) {
		lastActivity = lastProxied
	}
	if containerActive || holdActive {
		lastActivity = now
	}
	w.setLastActivity(lastActivity)
	idleDuration = now.Sub(lastActivity)
	return idleDuration, idleDuration >= w.idleTimeout
}

func (w *IdleWatcher) getLastActivity() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastActivity
}

func (w *IdleWatcher) setLastActivity(t time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lastActivity = t
}

// Status reports the current idle duration and whether dry-run mode is
// enabled, for the read-only /status diagnostic endpoint.
func (w *IdleWatcher) Status() (idleDuration time.Duration, dryRun bool) {
	return w.now().Sub(w.getLastActivity()), w.dryRun
}
