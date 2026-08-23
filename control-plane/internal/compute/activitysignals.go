package compute

import (
	"sync"
	"sync/atomic"
	"time"
)

const holdDuration = 30 * time.Minute

// ActivitySignals is the single source of truth for whether Compute
// currently counts as idle: hold state, last-proxied time, and the
// lastActivity clock plus the three-signal evaluation. Handler mutates hold
// and lastProxied through its HTTP endpoints and reads Status for
// GET /status; IdleWatcher reads it every poll tick. Constructed once in
// main.go and handed to both, so neither depends on the other.
type ActivitySignals struct {
	logger      EventLogger
	idleTimeout time.Duration
	dryRun      bool
	now         func() time.Time

	// lastProxied is written on every proxied HTTP request in Handler's
	// hot path -- deliberately a lock-free atomic, not folded under mu, so
	// a hold acquire/release never contends with request handling.
	lastProxied atomic.Int64 // unix nano; zero means never proxied

	mu           sync.Mutex
	lastActivity time.Time
	holdActive   bool
	holdExpires  time.Time
}

// NewActivitySignals resets lastActivity to now -- the process-start reset.
// The other reset (immediately after Suspend returns) happens via
// SetLastActivity inside IdleWatcher.poll, since suspend-to-RAM freezes the
// process rather than restarting it, so the process-start reset alone would
// never fire again on wake.
func NewActivitySignals(logger EventLogger, idleTimeout time.Duration, dryRun bool) *ActivitySignals {
	now := time.Now
	return &ActivitySignals{logger: logger, idleTimeout: idleTimeout, dryRun: dryRun, now: now, lastActivity: now()}
}

// Now lets IdleWatcher read the same clock Handler does, instead of each
// owning a separate one.
func (s *ActivitySignals) Now() time.Time {
	return s.now()
}

func (s *ActivitySignals) DryRun() bool {
	return s.dryRun
}

func (s *ActivitySignals) RecordProxied(t time.Time) {
	s.lastProxied.Store(t.UnixNano())
}

// LastProxiedAt returns the zero time if no request has ever been proxied.
func (s *ActivitySignals) LastProxiedAt() time.Time {
	nano := s.lastProxied.Load()
	if nano == 0 {
		return time.Time{}
	}
	return time.Unix(0, nano)
}

// HoldAcquire starts or renews the hold, always resetting the TTL to
// holdDuration from now.
func (s *ActivitySignals) HoldAcquire(identity string) {
	s.mu.Lock()
	s.holdActive = true
	s.holdExpires = s.now().Add(holdDuration)
	s.mu.Unlock()
	s.logger.SendEvent("hold_acquired", "acquired", &identity, nil)
}

// HoldRelease is a no-op (and logs nothing) if no hold is active, so a
// script can call it unconditionally on exit.
func (s *ActivitySignals) HoldRelease(identity string) {
	s.mu.Lock()
	wasActive := s.holdActive
	s.holdActive = false
	s.mu.Unlock()
	if wasActive {
		s.logger.SendEvent("hold_released", "released", &identity, nil)
	}
}

// checkExpiry is run wherever hold state is read (HoldActive,
// HoldRemainingTTL, Evaluate, Status), replacing the old hold.go's
// background time.AfterFunc + generation counter. IdleWatcher already ticks
// once a minute regardless of hold state, so expiry is noticed within one
// poll of its real deadline -- simpler than guarding a background timer
// against a stale fire after a renewal, at the cost of no longer being
// exact-on-time.
func (s *ActivitySignals) checkExpiry() {
	now := s.now()
	s.mu.Lock()
	due := s.holdActive && !now.Before(s.holdExpires)
	if due {
		s.holdActive = false
	}
	s.mu.Unlock()
	if due {
		s.logger.SendEvent("hold_expired", "expired", nil, nil)
	}
}

func (s *ActivitySignals) HoldActive() bool {
	s.checkExpiry()
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.holdActive
}

// HoldRemainingTTL returns zero if no hold is active.
func (s *ActivitySignals) HoldRemainingTTL() time.Duration {
	s.checkExpiry()
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.holdActive {
		return 0
	}
	if remaining := s.holdExpires.Sub(s.now()); remaining > 0 {
		return remaining
	}
	return 0
}

// Evaluate is IdleWatcher.poll's decision core: given the current
// container-activity signal, advances lastActivity and reports the current
// idle duration plus whether idleTimeout has been reached.
func (s *ActivitySignals) Evaluate(containerActive bool) (idleDuration time.Duration, trigger bool) {
	now := s.now()
	holdActive := s.HoldActive()

	s.mu.Lock()
	lastActivity := s.lastActivity
	s.mu.Unlock()

	if proxied := s.LastProxiedAt(); proxied.After(lastActivity) {
		lastActivity = proxied
	}
	if containerActive || holdActive {
		lastActivity = now
	}

	s.mu.Lock()
	s.lastActivity = lastActivity
	s.mu.Unlock()

	idleDuration = now.Sub(lastActivity)
	return idleDuration, idleDuration >= s.idleTimeout
}

func (s *ActivitySignals) SetLastActivity(t time.Time) {
	s.mu.Lock()
	s.lastActivity = t
	s.mu.Unlock()
}

func (s *ActivitySignals) Status() (idleDuration time.Duration, holdActive bool, holdRemainingTTL time.Duration, dryRun bool) {
	now := s.now()
	s.mu.Lock()
	lastActivity := s.lastActivity
	s.mu.Unlock()
	return now.Sub(lastActivity), s.HoldActive(), s.HoldRemainingTTL(), s.dryRun
}
