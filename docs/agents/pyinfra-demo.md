# pyinfra-provisioning: verification performed

Real, live verification performed for tickets whose acceptance criteria
can't be checked by a type checker or test suite alone — infrastructure
code's checks run through the tool's own execution engine or a real
adjacent tool, not mocks (the same "highest available seam" philosophy
`.scratch/pyinfra-provisioning/spec.md`'s Testing Decisions describes).

## Ticket 04 demo: `git_systemd_service` verified live

Ticket 04 (`.scratch/pyinfra-provisioning/issues/04-control-api-shared-deploy-helper.md`)
requires the shared git-pull-plus-systemd-unit helper
(`pyinfra/control_api_deploy.py`'s `git_systemd_service`) to be "demoed
against a throwaway app/unit before either real Control API Deploy file
adopts it." This was run for real, not assumed — no pytest/dry-run seam
applies to a pyinfra deploy function, so the only faithful check is
actually running it (the same "highest available seam" testing philosophy
`.scratch/pyinfra-provisioning/spec.md`'s Testing Decisions describes).

### Setup

- A throwaway bare git repo (`demo.git`) with a one-file "app" (`app.py`)
  that appends a line to a heartbeat file and sleeps.
- A scratch pyinfra deploy file exercising the exact same operation
  sequence and `.will_change`-gated restart logic as
  `control_api_deploy.py`'s `git_systemd_service` — `git.repo` →
  `files.template` → `systemd.service(restarted=code_changed or
  unit.will_change, daemon_reload=unit.will_change)` — run against `@local`
  with `user_mode=True` so it needs no root (the real Control APIs use
  system-mode units on `pi`/`server`; user-mode is a sandbox-only stand-in
  for exercising the identical logic without privileged access).
- Both used the real `templates/control-api.service.j2` unit template
  shape (minus the `User=` directive, which is invalid inside a user-mode
  unit — a systemd quirk specific to the demo's sandbox setup, not a
  concern for the real system-mode units `deploy_control_pi_api.py`/
  `deploy_control_server_api.py` install).

### Results

1. **First run** (fresh clone, no unit installed yet): `git.repo` and
   `files.template` both report `Success` (real clone, real file write),
   `systemd.service` enables and starts the unit. `systemctl --user
   status` confirms it `Active: running`; the heartbeat file gets a real
   `beat v1` line.
2. **Second run, nothing changed**: all three operations report `No
   Change` — `Grand total: 3 success=0, no_change=3` — zero pending
   operations, confirming the pattern is genuinely idempotent, not merely
   "ran without error once."
3. **Third run, after pushing a code-only change** (`beat v1` →
   `beat v2` in `app.py`, new commit pushed to the demo repo, unit file
   untouched): `git.repo` reports `Success` (pulled the new commit),
   `files.template` reports `No Change` (unit file identical), and
   `systemd.service` still reports `Success` — proving `restarted=
   code_changed or unit.will_change` correctly restarts on a **code-only**
   change even though the unit file itself didn't change. `systemctl
   --user status` shows a new PID and a later `Active since` timestamp;
   the heartbeat file gained a `beat v2` line from the new process,
   confirming the running code actually changed, not just that some
   command exited 0.

This is exactly ticket 04's required behavior — "restarts the service
when a Deploy changes the pulled code or the unit file, and leaves it
alone otherwise" — demonstrated end to end before
`deploy_control_pi_api.py` (ticket 05) and `deploy_control_server_api.py`
(ticket 06) adopted the same helper.

### A note on `.will_change` vs `.did_change()`

pyinfra's `OperationMeta.did_change()` is a **method** that raises
(`Cannot evaluate operation result before execution`) if read before the
operation has actually executed — safe only well after the point in a
deploy file where you'd naturally write `restarted=repo.did_change`. The
property that's safe to read immediately after calling an operation
(during the prepare-phase diff, before execution) is `.will_change` —
confirmed by reading `pyinfra/api/operation.py`'s `OperationMeta` class in
the installed `pyinfra==3.9.2` package, and by this demo actually running
without the `RuntimeError` that `.did_change()` would raise in the same
position. `control_api_deploy.py` uses `.will_change` throughout for this
reason.

## Ticket 07: templated Caddyfile validated with a real `caddy` binary

Ticket 07's `templates/Caddyfile.j2` (rendered by `deploy_caddy.py`) was
checked with the actual `caddy validate --config ... --adapter caddyfile`
command against a real Caddy binary (`caddy_2.11.4_linux_amd64`,
downloaded fresh from Caddy's GitHub releases in this sandbox — no
`caddy`/`go`/`xcaddy` preinstalled), the same tool
`services/auto_wake_proxy/README.md` used to validate the original
(now-templated) Caddyfile this file evolved from:

1. Rendered `templates/Caddyfile.j2` standalone (plain Jinja2, the values
   `deploy_caddy.py` would pass: two `WORKLOADS` entries, the Control UI
   root/port, the Control Pi API port) to get real output, not inspected
   source.
2. `caddy validate` against that rendered file with a **stock** binary
   (no plugins compiled in) fails with exactly one error: `wake_on_lan is
   not a registered directive` — expected and correct, since `wake_on_lan`
   ships in the third-party `github.com/dulli/caddy-wol` plugin
   (`services/auto_wake_proxy/README.md`'s Plugins section), not stock
   Caddy. This is the same finding that README already documented for the
   un-templated original file.
3. With the two `wake_on_lan`-dependent lines stripped (the `order
   wake_on_lan before respond` global option and the `wake_on_lan {...}`
   call inside `handle_errors`), the **same stock binary reports `Valid
   configuration`** — confirming every directive this ticket actually
   added (the `{% for workload in workloads %}` site-block loop, and the
   new Control UI/Control Pi API site's `handle`/`reverse_proxy`/
   `root`/`file_server` block) is genuinely valid Caddyfile syntax, not
   just "looks right." The only unvalidated piece is the plugin-gated
   directive this file inherited unchanged from the already-verified
   original.

Building the `caddy-wol`-enabled binary itself (`xcaddy build --with
github.com/dulli/caddy-wol`) was not repeated here — it's already real,
researched, and verified in `services/auto_wake_proxy/README.md`, and
`deploy_caddy.py`'s docstring is explicit that provisioning that binary is
outside this Deploy file's scope.

## caddy-wake-plugin: `wake_plugin` validated live, `caddy-wol` retired

ADR-0012 replaced `caddy-wol` with the in-repo `services/auto_wake_proxy/wake_plugin`
module (the `call_wake_api` directive, calling the Control Pi API's
`/wake` instead of sending its own WoL packet). Unlike the `caddy-wol`
line above, this one *was* actually compiled in and exercised:

1. Built a real Caddy binary with `xcaddy build --with
   github.com/nikbogman/homelab/services/auto_wake_proxy/wake_plugin=./services/auto_wake_proxy/wake_plugin`
   (local module override — this repo isn't fetchable as a public Go
   module yet).
2. `caddy validate` against `templates/Caddyfile.j2` rendered with
   representative values reported `Valid configuration` with that binary.
3. A live `caddy run` against that config, a fake Control Pi API (a
   throwaway HTTP server logging `POST /wake`), and a deliberately
   unreachable upstream: the first request to `/server*` triggered a
   `POST /wake` and returned 502 once the retry window elapsed; a second
   request within the throttle window skipped the `/wake` call
   ("throttled, skipping") and still returned 502 the same way — matching
   the fail-open, throttled design in `.scratch/caddy-wake-plugin/spec.md`.

Step 2 is what surfaced the `handle_errors`-inside-`handle_path` bug
documented in `pyinfra/README.md`'s "What was actually verified" section
— found and fixed here, not carried over from `caddy-wol`.
