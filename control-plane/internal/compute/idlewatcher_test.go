package compute

import (
	"errors"
	"testing"
	"time"
)

const testIdleTimeout = time.Hour

// mustNewIdleWatcher wires an IdleWatcher whose clock, hold state, and
// last-proxied time are all test-controlled -- nothing here ever sleeps
// for a real poll interval or idle timeout.
func mustNewIdleWatcher(t *testing.T, suspender Suspender, runtime ContainerRuntime, logger EventLogger, start time.Time, holdActive func() bool, lastProxied func() time.Time) *IdleWatcher {
	t.Helper()
	return &IdleWatcher{
		suspender:    suspender,
		runtime:      runtime,
		logger:       logger,
		holdActive:   holdActive,
		lastProxied:  lastProxied,
		now:          func() time.Time { return start },
		idleTimeout:  testIdleTimeout,
		lastActivity: start,
	}
}

func noHold() bool            { return false }
func neverProxied() time.Time { return time.Time{} }

func TestContainerActivityAboveBaselineNeverTriggersSuspendHoweverLongItPolls(t *testing.T) {
	start := time.Now()
	suspender := &fakeSuspender{}
	runtime := &fakeContainerRuntime{active: true}
	w := mustNewIdleWatcher(t, suspender, runtime, &fakeLogger{}, start, noHold, neverProxied)

	// Poll well past the idle timeout, container activity staying above baseline throughout.
	for i := 0; i < 5; i++ {
		w.now = func() time.Time { return start.Add(time.Duration(i) * 2 * testIdleTimeout) }
		w.poll()
	}

	if suspender.calls != 0 {
		t.Fatalf("got %d suspend calls, want 0: container activity should have suppressed every poll", suspender.calls)
	}
}

func TestRecentProxiedRequestWithNoContainerActivityNeverTriggersSuspend(t *testing.T) {
	start := time.Now()
	suspender := &fakeSuspender{}
	runtime := &fakeContainerRuntime{active: false}
	lastProxied := start.Add(30 * time.Minute)
	w := mustNewIdleWatcher(t, suspender, runtime, &fakeLogger{}, start, noHold, func() time.Time { return lastProxied })

	// Just past the idle timeout measured from process start, but well
	// within it measured from the more recent proxied request.
	w.now = func() time.Time { return start.Add(testIdleTimeout + time.Minute) }
	w.poll()

	if suspender.calls != 0 {
		t.Fatalf("got %d suspend calls, want 0: recent proxied request should have suppressed suspend", suspender.calls)
	}
}

func TestActiveHoldNeverTriggersSuspendEvenPastTheIdleTimeout(t *testing.T) {
	start := time.Now()
	suspender := &fakeSuspender{}
	runtime := &fakeContainerRuntime{active: false}
	w := mustNewIdleWatcher(t, suspender, runtime, &fakeLogger{}, start, func() bool { return true }, neverProxied)

	for i := 0; i < 5; i++ {
		w.now = func() time.Time { return start.Add(time.Duration(i) * 2 * testIdleTimeout) }
		w.poll()
	}

	if suspender.calls != 0 {
		t.Fatalf("got %d suspend calls, want 0: an active hold should have suppressed every poll", suspender.calls)
	}
}

func TestNoActivitySignalForAFullHourTriggersTheSameSuspender(t *testing.T) {
	start := time.Now()
	suspender := &fakeSuspender{}
	runtime := &fakeContainerRuntime{active: false}
	w := mustNewIdleWatcher(t, suspender, runtime, &fakeLogger{}, start, noHold, neverProxied)

	w.now = func() time.Time { return start.Add(testIdleTimeout) }
	w.poll()

	if suspender.calls != 1 {
		t.Fatalf("got %d suspend calls, want 1", suspender.calls)
	}
}

func TestNoActivitySignalBeforeTheIdleTimeoutDoesNotTrigger(t *testing.T) {
	start := time.Now()
	suspender := &fakeSuspender{}
	runtime := &fakeContainerRuntime{active: false}
	w := mustNewIdleWatcher(t, suspender, runtime, &fakeLogger{}, start, noHold, neverProxied)

	w.now = func() time.Time { return start.Add(testIdleTimeout - time.Minute) }
	w.poll()

	if suspender.calls != 0 {
		t.Fatalf("got %d suspend calls, want 0: one minute short of the idle timeout", suspender.calls)
	}
}

func TestAutoSuspendLogsTriggeredThenSucceededWithNilIdentityAndIdleDuration(t *testing.T) {
	start := time.Now()
	logger := &fakeLogger{}
	runtime := &fakeContainerRuntime{active: false}
	w := mustNewIdleWatcher(t, &fakeSuspender{}, runtime, logger, start, noHold, neverProxied)

	w.now = func() time.Time { return start.Add(testIdleTimeout) }
	w.poll()

	if len(logger.events) != 2 {
		t.Fatalf("got %d events, want 2: %+v", len(logger.events), logger.events)
	}
	first, second := logger.events[0], logger.events[1]
	if first.eventType != "suspend_auto_triggered" || first.outcome != "triggered" || first.identity != nil {
		t.Fatalf("got first event %+v, want suspend_auto_triggered/triggered/nil", first)
	}
	if second.eventType != "suspend_auto_succeeded" || second.outcome != "succeeded" || second.identity != nil {
		t.Fatalf("got second event %+v, want suspend_auto_succeeded/succeeded/nil", second)
	}
	for _, e := range logger.events {
		if e.extra == nil || e.extra["idleSeconds"] == nil {
			t.Fatalf("got event %+v, want idleSeconds recorded in extra", e)
		}
	}

	// Distinct from, and doesn't touch, the manual event types.
	for _, e := range logger.events {
		if e.eventType == "suspend_requested" || e.eventType == "suspend_succeeded" || e.eventType == "suspend_failed" {
			t.Fatalf("got manual event type %q from the automatic path, want only suspend_auto_*", e.eventType)
		}
	}
}

func TestAutoSuspendLogsFailedWithNilIdentityWhenSuspenderFails(t *testing.T) {
	start := time.Now()
	logger := &fakeLogger{}
	suspender := &fakeSuspender{err: errors.New("systemctl suspend: exit status 1")}
	runtime := &fakeContainerRuntime{active: false}
	w := mustNewIdleWatcher(t, suspender, runtime, logger, start, noHold, neverProxied)

	w.now = func() time.Time { return start.Add(testIdleTimeout) }
	w.poll()

	last := logger.events[len(logger.events)-1]
	if last.eventType != "suspend_auto_failed" || last.outcome != "failed" || last.identity != nil {
		t.Fatalf("got last event %+v, want suspend_auto_failed/failed/nil", last)
	}
}

func TestLastActivityResetsAtConstructionTime(t *testing.T) {
	start := time.Now()
	suspender := &fakeSuspender{}
	runtime := &fakeContainerRuntime{active: false}
	w := NewIdleWatcher(suspender, runtime, &fakeLogger{}, noHold, neverProxied, testIdleTimeout)

	if w.lastActivity.Before(start) {
		t.Fatalf("got lastActivity %v, want it reset to construction time (>= %v)", w.lastActivity, start)
	}
}

func TestLastActivityResetsImmediatelyAfterSuspendReturns(t *testing.T) {
	start := time.Now()
	suspender := &fakeSuspender{}
	runtime := &fakeContainerRuntime{active: false}
	w := mustNewIdleWatcher(t, suspender, runtime, &fakeLogger{}, start, noHold, neverProxied)

	wakeTime := start.Add(testIdleTimeout + 5*time.Hour) // stands in for time elapsed while suspended
	w.now = func() time.Time { return wakeTime }
	w.poll()

	if suspender.calls != 1 {
		t.Fatalf("got %d suspend calls, want 1", suspender.calls)
	}
	if !w.lastActivity.Equal(wakeTime) {
		t.Fatalf("got lastActivity %v, want it reset to the post-Suspend wake time %v", w.lastActivity, wakeTime)
	}

	// Immediately re-polling at the same wake time must not immediately
	// re-trigger -- the whole point of the post-Suspend reset.
	suspender.calls = 0
	w.poll()
	if suspender.calls != 0 {
		t.Fatalf("got %d suspend calls on the very next poll after waking, want 0", suspender.calls)
	}
}
