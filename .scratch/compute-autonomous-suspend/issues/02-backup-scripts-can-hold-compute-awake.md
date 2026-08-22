# 02 — Backup/restore scripts can hold Compute awake during their work

**What to build:** A Hold API lets a long-running operation (backup/restore) prevent Compute from suspending out from under it. `POST /hold` acquires a fixed 30-minute hold (not caller-specifiable); calling it again before expiry renews it. `DELETE /hold` releases it early. A hold that isn't renewed simply expires — no cleanup step required. Both endpoints accept either the standard tailnet identity header or a loopback (127.0.0.1) caller in its place, so an unattended script running locally on Compute needs neither a browser session nor a tailnet identity — reuse the existing loopback-auth pattern already used elsewhere in this codebase rather than writing new auth logic.

**Blocked by:** None — can start immediately.

**Status:** ready-for-human

- [x] `POST /hold` acquires a 30-minute hold; calling it again before expiry renews the TTL rather than stacking holds.
- [x] `DELETE /hold` releases an active hold immediately.
- [x] Both endpoints accept the tailnet identity header OR a loopback caller — rejecting neither, requiring at least one.
- [x] An unrenewed hold expires on its own after 30 minutes with no separate cleanup call.
- [x] Hold acquire/release/expire each produce a distinct event log entry.

**Implementation notes:**
- Reused `tailnet.RequireTailnetIdentityOrLoopback` as-is (already existed for Gateway's `/wake`); no new auth logic.
- Hold state (`control-plane/internal/compute/hold.go`) tracks only `active bool` + a `generation` counter, not the actual `*time.Timer` — a stale timer is left to fire and no-op via a generation mismatch rather than being cancelled, since `time.Timer.Stop()` can't guarantee a fired goroutine hasn't already started. Simpler than storing/cancelling timers and race-free.
- Renewal logs another `hold_acquired`, not a separate `hold_renewed` event type — the spec asks for three distinct event *categories* (acquire/release/expire), not a fourth.
- `DELETE /hold` with no active hold is a no-op that logs nothing (still returns 200), so a script can call it unconditionally on exit without producing a misleading `hold_released` event for a hold that was never there.
- No `IsHeld`/`Remaining` read surface exposed yet — deferred to ticket 04 (`GET /status`), which is the first actual consumer; ticket 03's idle watcher doesn't exist yet either, so nothing reads hold state today.
