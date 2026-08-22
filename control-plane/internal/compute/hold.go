package compute

import (
	"sync"
	"time"
)

// Package var, not const, so tests can shrink it.
var holdDuration = 30 * time.Minute

// hold is a caller-renewable lease that will let the future idle watcher
// know a long-running script (e.g. backup/restore) is in progress. generation
// guards against a stale timer firing after a renewal or release replaces
// it -- time.AfterFunc gives no way to know whether a fired callback has
// already started before a later acquire/release runs, so staleness is
// checked inside the callback instead of trying to cancel it.
type hold struct {
	mu         sync.Mutex
	active     bool
	generation int
}

// acquire starts or renews the hold, always resetting the TTL to
// holdDuration from now.
func (h *hold) acquire(logger EventLogger, identity string) {
	h.mu.Lock()
	h.active = true
	h.generation++
	gen := h.generation
	h.mu.Unlock()

	logger.SendEvent("hold_acquired", "acquired", &identity, nil)
	time.AfterFunc(holdDuration, func() { h.expire(logger, gen) })
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
