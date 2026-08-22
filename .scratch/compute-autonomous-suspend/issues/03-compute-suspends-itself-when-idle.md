# 03 — Compute suspends itself after an hour of no real use

**What to build:** The idle watcher runs continuously inside Compute API: once a minute it evaluates whether Compute is idle — no workload container CPU/network activity above baseline, no request proxied to a workload within the current window, and no active Hold — and after a full hour of that being continuously true, triggers the exact same `Suspender.Suspend()` the manual Suspend button already calls. Manual suspend, its request/response contract, and its existing event types are untouched; this is additive. Automatic triggers/successes/failures log under their own event types with no human identity attached, idle duration recorded. Last-activity resets both at process start and immediately when `Suspender.Suspend()` returns — the latter because suspend-to-RAM freezes the process rather than restarting it, so a process-start-only reset would never fire on wake, and Compute would otherwise be one poll cycle away from immediately re-suspending itself right after waking.

**Blocked by:** 01 (Compute routes workload requests to Docker containers by label), 02 (Backup/restore scripts can hold Compute awake during their work)

**Status:** ready-for-agent

- [x] With container activity above baseline, the watcher never triggers suspend, however long it polls.
- [x] With a recent proxied request but no container CPU/network activity, the watcher never triggers suspend.
- [x] With an active Hold, the watcher never triggers suspend even past the idle timeout.
- [x] With none of the three activity signals true for a full hour, the watcher calls the same `Suspender.Suspend()` the manual `/suspend` handler uses.
- [x] The idle-watcher decision logic is tested as a pure function against fake activity/hold/clock inputs — no test actually sleeps for the real poll interval or idle timeout.
- [x] Last-activity is reset at process start and again immediately after `Suspender.Suspend()` returns.
- [x] Automatic suspend triggered/succeeded/failed events are logged under their own event types, with a nil identity and idle duration recorded — distinct from the manual `suspend_requested`/`_succeeded`/`_failed` types, which are unchanged.

## Comments

Implemented: `internal/compute/idlewatcher.go` (`IdleWatcher`, decision logic in `evaluate`/`poll`), wired in `cmd/compute-api/main.go`. `ContainerRuntime` gained `ActivityAboveBaseline()`; `DockerRuntime`'s implementation is CPU-only for now (network-delta signal deferred, marked with a `ponytail:` comment in `docker.go` — nothing routable today needs it). CPU baseline is env-configurable (`COMPUTE_API_CPU_BASELINE_PERCENT`, default 2%) per the spec's operating-parameters table. Dry-run mode and `GET /status` are issue 04, not touched here.
