# 01 — Compute routes workload requests to Docker containers by label

**What to build:** Compute API becomes its own reverse proxy. A workload container becomes routable simply by carrying a `homelab.route=/path` label in its compose file — no compute-api code change or redeploy. A request to `/{prefix}/...` on Compute API routes to whichever container carries that label. Introduces the `ContainerRuntime` seam (Docker daemon access) that both this proxy and the future idle watcher depend on: listing routable containers by label, and per-container CPU/network activity data (the latter unused until the idle-watcher ticket, but part of the same seam). Compute API talks to the Docker daemon directly — no sidecar reverse proxy, no static routing config.

**Blocked by:** None — can start immediately.

**Status:** ready-for-human

- [x] A container labeled `homelab.route=/immich` (or similar) is discovered automatically and a request to `/immich/...` is proxied to it, with no compute-api config change.
- [x] A container carrying no `homelab.route` label is never routed to.
- [x] A real HTTP error from a live backend container passes through untouched (mirrors the Gateway proxy's existing behavior).
- [x] `ContainerRuntime` is injected into the handler the same way `Suspender`/`EventLogger` already are, with a fake implementation used for all proxy tests — no real Docker daemon required in tests.
- [x] The proxy records when a request was last successfully proxied to a workload container, in a form the idle watcher can later consume as an activity signal.

**Implementation notes:**
- `ContainerRuntime` currently exposes only `RoutableContainers()` — the spec's mention of a per-container CPU/network activity method was deferred to the idle-watcher ticket rather than built speculatively now with no caller; that ticket can extend the interface once it has a real consumer to design the method's shape against.
- Uses the current official Docker Go SDK (`github.com/moby/moby/client` — `github.com/docker/docker/client` is the deprecated predecessor path), latest v0.5.1, which only requires Go 1.24 already in use.
- A labeled container is only routable if it publishes a port to the host (`127.0.0.1:PublicPort`); one lacking a published port is skipped with a log line, not silently dropped.
