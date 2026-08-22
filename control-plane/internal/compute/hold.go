package compute

import (
	"sync"
	"time"
)

// Package var, not const, so tests can shrink it.
var holdDuration = 30 * time.Minute

// hold is a caller-renewable lease that lets the idle watcher know a
// long-running script (e.g. backup/restore) is in progress. generation
// guards against a stale timer firing after a renewal or release replaces
// it -- time.AfterFunc gives no way to know whether a fired callback has
// already started before a later acquire/release runs, so staleness is
// checked inside the callback instead of trying to cancel it.
type hold struct {
	mu         sync.Mutex
	active     bool
	generation int
	expiresAt  time.Time
}

// acquire starts or renews the hold, always resetting the TTL to
// holdDuration from now.
func (h *hold) acquire(logger EventLogger, identity string) {
	h.mu.Lock()
	h.active = true
	h.generation++
	gen := h.generation
	h.expiresAt = time.Now().Add(holdDuration)
	h.mu.Unlock()

	logger.SendEvent("hold_acquired", "acquired", &identity, nil)
	time.AfterFunc(holdDuration, func() { h.expire(logger, gen) })
}

// isActive reports whether a hold is currently in effect -- the idle
// watcher's third suppression signal, alongside container activity and
// recent proxy traffic.
func (h *hold) isActive() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.active
}

// remainingTTL reports time left before the active hold expires, or zero
// if no hold is active -- the idle watcher validation status endpoint's
// hold-TTL field.
func (h *hold) remainingTTL() time.Duration {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.active {
		return 0
	}
	if remaining := time.Until(h.expiresAt); remaining > 0 {
		return remaining
	}
	return 0
}

// release ends an active hold immediately. It's a no-op (and logs nothing)
// if no hold is active, so a script can call it unconditionally on exit.
func (h *hold) release(logger EventLogger, identity string) {
	h.mu.Lock()
	wasActive := h.active
	h.active = false
	h.generation++
	h.mu.Unlock()

	if wasActive {
		logger.SendEvent("hold_released", "released", &identity, nil)
	}
}

// expire is the timer callback from acquire. No identity: nothing called
// this, it fired on its own.
func (h *hold) expire(logger EventLogger, gen int) {
	h.mu.Lock()
	if !h.active || h.generation != gen {
		h.mu.Unlock()
		return
	}
	h.active = false
	h.mu.Unlock()

	logger.SendEvent("hold_expired", "expired", nil, nil)
}
