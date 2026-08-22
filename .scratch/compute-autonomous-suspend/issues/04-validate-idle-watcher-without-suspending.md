# 04 — Idle-watcher behavior can be validated without actually suspending the machine

**What to build:** A dry-run mode and a status endpoint make the idle watcher observable and safely testable on a real box. An env var, disabled by default, leaves the watcher's real polling/timer/activity-signal evaluation completely unchanged — including Hold suppression — but on trigger logs a distinct dry-run event and resets last-activity instead of calling `Suspender.Suspend()`, so the watcher keeps cycling and can be observed firing repeatedly in one session. A read-only status endpoint reports current idle duration, whether a Hold is active (and its remaining TTL), and whether dry-run is currently enabled.

**Blocked by:** 03 (Compute suspends itself after an hour of no real use)

**Status:** ready-for-agent

- [x] With dry-run enabled, reaching the idle timeout logs the dry-run event and does not call `Suspender.Suspend()`.
- [x] With dry-run enabled, an active Hold still suppresses the trigger exactly as it does outside dry-run.
- [x] With dry-run enabled, the watcher keeps cycling after a trigger rather than stopping — a second idle-timeout period produces a second dry-run trigger.
- [x] `GET /status` reports idle duration, active-hold state with remaining TTL, and the current dry-run flag.
- [x] `GET /status` requires the standard tailnet identity header — no loopback exception, since it is a read diagnostic, not an automation target.

## Comments

Implemented: `IdleWatcher` gained a `dryRun` field (`internal/compute/idlewatcher.go`) — `poll()` runs the exact same `evaluate()` decision path either way, branching only at the final call site: dry-run logs `suspend_auto_dry_run` and resets last-activity instead of invoking `Suspender.Suspend()`. `lastActivity` is now mutex-guarded (`getLastActivity`/`setLastActivity`) since `GET /status` reads it concurrently with the watcher's own poll goroutine. `hold` gained `expiresAt` + `remainingTTL()` (`internal/compute/hold.go`) for the status endpoint's hold-TTL field. `Handler` gained `GET /status` (`RequireTailnetIdentity`, no loopback exception) and a `SetIdleStatus` setter, wired in `cmd/compute-api/main.go` after both `Handler` and `IdleWatcher` exist (breaks their construction cycle — `IdleWatcher` depends on `Handler.HoldActive`/`LastProxiedAt`). Env var `COMPUTE_API_AUTOSUSPEND_DRY_RUN` (bool, default `false`) parsed via `strconv.ParseBool`, matching the ADR's operating-parameters table. Code-reviewed clean on both Standards and Spec axes; committed as `18b05c0`.
