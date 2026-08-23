package compute

import (
	"log"
	"time"
)

// IdleWatcher runs continuously inside Compute API, polling once a minute
// for whether Compute counts as idle via ActivitySignals: no workload
// container CPU/network activity above baseline, no request proxied to a
// workload within the current idle window, and no active Hold. After
// idleTimeout with none of those true, it triggers the exact same
// Suspender.Suspend() the manual /suspend handler calls -- automatic and
// manual suspend are two callers of one code path, not two implementations.
type IdleWatcher struct {
	suspender Suspender
	runtime   ContainerRuntime
	logger    EventLogger
	signals   *ActivitySignals
}

func NewIdleWatcher(suspender Suspender, runtime ContainerRuntime, logger EventLogger, signals *ActivitySignals) *IdleWatcher {
	return &IdleWatcher{suspender: suspender, runtime: runtime, logger: logger, signals: signals}
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
	now := w.signals.Now()

	containerActive, err := w.runtime.ActivityAboveBaseline()
	if err != nil {
		// Fail toward staying awake, not toward suspending on a Docker
		// hiccup: an unreadable signal is treated as activity.
		log.Printf("idle watcher: checking container activity: %v", err)
		containerActive = true
	}

	idleDuration, trigger := w.signals.Evaluate(containerActive)
	if !trigger {
		return
	}

	extra := map[string]any{"idleSeconds": idleDuration.Seconds()}

	if w.signals.DryRun() {
		// Same trigger, same decision logic -- only the final call differs.
		// Reset last-activity so the watcher keeps cycling instead of
		// re-triggering on every poll for the rest of the session.
		w.logger.SendEvent("suspend_auto_dry_run", "dry_run", nil, extra)
		w.signals.SetLastActivity(now)
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
	w.signals.SetLastActivity(w.signals.Now())
}
