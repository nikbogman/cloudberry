# 04 — Idle-watcher behavior can be validated without actually suspending the machine

**What to build:** A dry-run mode and a status endpoint make the idle watcher observable and safely testable on a real box. An env var, disabled by default, leaves the watcher's real polling/timer/activity-signal evaluation completely unchanged — including Hold suppression — but on trigger logs a distinct dry-run event and resets last-activity instead of calling `Suspender.Suspend()`, so the watcher keeps cycling and can be observed firing repeatedly in one session. A read-only status endpoint reports current idle duration, whether a Hold is active (and its remaining TTL), and whether dry-run is currently enabled.

**Blocked by:** 03 (Compute suspends itself after an hour of no real use)

**Status:** ready-for-agent

- [ ] With dry-run enabled, reaching the idle timeout logs the dry-run event and does not call `Suspender.Suspend()`.
- [ ] With dry-run enabled, an active Hold still suppresses the trigger exactly as it does outside dry-run.
- [ ] With dry-run enabled, the watcher keeps cycling after a trigger rather than stopping — a second idle-timeout period produces a second dry-run trigger.
- [ ] `GET /status` reports idle duration, active-hold state with remaining TTL, and the current dry-run flag.
- [ ] `GET /status` requires the standard tailnet identity header — no loopback exception, since it is a read diagnostic, not an automation target.
