# pyinfra 3.x reference

Ground-truth reference for implementing `.scratch/pyinfra-provisioning/issues/01-*.md` through `09-*.md` against **pyinfra 3.x** (verified against the installed `pyinfra==3.9.2` package and https://docs.pyinfra.com/en/3.x/). pyinfra 3.x is a rewrite of 1.x/2.x — do not trust blog posts, Stack Overflow, or general LLM training-data recall about pyinfra. Every claim below is cited to a docs.pyinfra.com/en/3.x/ page, or marked as derived from reading the installed package source when the docs prose doesn't spell it out.

## Vocabulary mapping

This repo's spec (`.scratch/pyinfra-provisioning/spec.md`) uses its own terms. Map them like this when reading pyinfra's docs/API:

| This repo's term | pyinfra's actual term | Notes |
|---|---|---|
| **Deploy** (a single run against the inventory) | pyinfra just calls this "running pyinfra" / "a deploy run". The CLI invocation `pyinfra inventory.py deploy.py ...` *is* the deploy. There's no separate noun for it beyond "CLI run". | [Getting Started](https://docs.pyinfra.com/en/3.x/getting-started.html) |
| **Deploy file** (single-responsibility, e.g. `deploy_tailscale.py`) — formerly called "Concern" in this repo's docs, now renamed to match pyinfra's own term | Same as pyinfra's own **deploy file**: a file containing operation calls, optionally wrapped in an `@deploy`-decorated **deploy function** for reuse/composition | [Using Operations](https://docs.pyinfra.com/en/3.x/using-operations.html), [Writing Deploys](https://docs.pyinfra.com/en/3.x/api/deploys.html) |
| **Host group** | **Group** — a named list of hosts in `inventory.py`, or a `@connector` prefix that generates hosts | [Inventory & Data](https://docs.pyinfra.com/en/3.x/inventory-data.html) |
| shared git+systemd helper | An `@deploy`-decorated function taking parameters, imported and called from multiple deploy files | [Writing Deploys](https://docs.pyinfra.com/en/3.x/api/deploys.html) |
| "--check dry run" (spec/ticket wording) | **`--dry`** — pyinfra 3.x has **no `--check` flag**. See Gotchas. | [Using the CLI](https://docs.pyinfra.com/en/3.x/cli.html) |

## Core concepts

- **Inventory**: the set of hosts + groups + data, defined in `inventory.py`. [Inventory & Data](https://docs.pyinfra.com/en/3.x/inventory-data.html)
- **Groups**: named lists of hosts (top-level list/tuple variables in `inventory.py`); a host can belong to multiple groups; group names cannot start with `_`. [Inventory & Data](https://docs.pyinfra.com/en/3.x/inventory-data.html)
- **Hosts**: individual targets — a hostname string, an `(name, data_dict)` tuple, or a `@connector/...` string. [Getting Started](https://docs.pyinfra.com/en/3.x/getting-started.html)
- **Operations**: idempotent, declarative units of state (`apt.packages(...)`, `systemd.service(...)`, `files.template(...)`) called inside a deploy file. [Using Operations](https://docs.pyinfra.com/en/3.x/using-operations.html)
- **Facts**: read-only state queries against a host (`server.Os`, `files.File`), gathered during the prepare phase and usable inside deploy code via `host.get_fact(...)`. [Facts](https://docs.pyinfra.com/en/3.x/facts.html)
- **Deploy functions**: reusable, parameterized bundles of operations wrapped with `@deploy(...)`, importable across deploy files. [Writing Deploys](https://docs.pyinfra.com/en/3.x/api/deploys.html)
- **Connectors**: pluggable transports that generate inventory hosts and/or execute commands — `@ssh` (default), `@local`, `@docker`, `@dockerssh`, `@terraform`, `@vagrant`, `@podman`/`@podmanssh`, `@chroot`. [Connectors](https://docs.pyinfra.com/en/3.x/connectors.html)
- **State**: the internal object tracking the deploy's operations/facts/results across hosts; you generally don't touch it directly (`host` is the main surface).

## Project file layout

pyinfra has **no enforced discovery convention** — `inventory.py` and `deploy.py` are pure convention, not required filenames. The CLI takes explicit filenames as positional args: `pyinfra INVENTORY OPERATIONS...`. [Getting Started](https://docs.pyinfra.com/en/3.x/getting-started.html), confirmed via `pyinfra --help` (installed 3.9.2).

For this repo, put everything under `pyinfra/`:
```
pyinfra/
  inventory.py                    # pi / server / test groups
  deploy_tailscale.py             # Concern (02)
  deploy_docker.py                # Concern (03)
  control_api_deploy.py           # shared @deploy helper (04)
  deploy_control_pi_api.py        # Concern (05)
  deploy_control_server_api.py    # Concern (06)
  deploy_caddy.py                 # Concern (07)
  templates/
    Caddyfile.j2
  deploy.py                       # entrypoint (08)
```
An optional `config.py` next to `inventory.py` can hold global config like `INHERIT_ENV`; the CLI looks for it via `--config` (default `config.py`) — confirmed via `pyinfra --help`.

## inventory.py syntax

Groups are top-level variables holding a list of host strings/tuples, or `(list, data_dict)` for shared group data. [Inventory & Data](https://docs.pyinfra.com/en/3.x/inventory-data.html)

```python
# pyinfra/inventory.py

pi = [
    ("pi-zero.tailnet", {"ssh_user": "pi"}),
]

server = [
    ("main-server.tailnet", {"ssh_user": "admin"}),
]

# Disposable containers standing in for pi/server OS bases (ticket 09).
# See "Connectors: @docker" below for what this can and cannot exercise.
test = [
    "@docker/debian:bookworm",
]
```

Group-wide data can also live in `group_data/<group_name>.py` (a plain Python file whose module-level names become that group's data), with `group_data/all.py` applied to every host. Precedence, lowest to highest: `group_data/all.py` → `group_data/<group>.py` → inline host data in `inventory.py` → CLI `--data`/`--group-data` overrides. [Inventory & Data](https://docs.pyinfra.com/en/3.x/inventory-data.html)

Verified working (`pyinfra inventory.py fact server.Os --limit test`, real run against the demo inventory above) — `--limit <group-name>` targets a single group defined in `inventory.py`.

## Operation basics

Operations are imported from `pyinfra.operations.<module>` and called as functions inside a deploy file; the `name=` kwarg is the human-readable label shown in output. [Using Operations](https://docs.pyinfra.com/en/3.x/using-operations.html)

```python
from pyinfra.operations import apt

apt.packages(
    name="Ensure vim is installed",
    packages=["vim"],
    update=True,
)
```

**`host.data`** exposes per-host/group data from the inventory:
```python
from pyinfra import host
from pyinfra.operations import git

git.repo(
    name="Clone app",
    src="git@github.com:user/repo.git",
    dest=host.data.app_dir,
)
```
`host.groups` (list of group names the current host belongs to) and `host.name` are also available for branching. [Using Operations](https://docs.pyinfra.com/en/3.x/using-operations.html) — confirmed against source (`pyinfra/api/host.py`, `self.groups = groups`).

**Execution model — two phases**, per host, in lock-step across hosts: [Deploy Process](https://docs.pyinfra.com/en/3.x/deploy-process.html)
1. **Prepare**: your deploy file's Python runs once per host to determine which operations apply and in what order; facts are read but nothing is mutated.
2. **Execute**: pyinfra walks operations in order; for operation N, it re-evaluates and runs on every host in parallel, and *all* hosts finish operation N before any host starts N+1.

Because prepare-phase Python conditionals run before any operation executes, you cannot branch in plain Python on "did the previous operation change something" — use the `_if` global argument for execute-time conditionals instead. Facts read during prepare are also a prepare-time snapshot; the docs explicitly warn "only use immutable facts in deploy code (installed OS, Arch, etc) unless you are absolutely sure they will not change." [Deploy Process](https://docs.pyinfra.com/en/3.x/deploy-process.html)

## files.template (Caddyfile — ticket 07)

Signature (from the installed 3.9.2 package, matches [Files operations](https://docs.pyinfra.com/en/3.x/operations/files.html)):
```python
files.template(
    src: str | IO,
    dest: str,
    user: str | None = None,
    group: str | None = None,
    mode: str | None = None,
    create_remote_dir: bool = True,
    jinja_env_kwargs: dict | None = None,
    **data,
)
```
- `host`, `state`, and `inventory` are injected into the Jinja2 template context automatically. Anything else must be passed explicitly as extra kwargs (`**data`), which become template variables. [Files operations](https://docs.pyinfra.com/en/3.x/operations/files.html)
- Convention: templates live in a `templates/` dir with a `.j2` suffix. Jinja2 `extends`/`import`/`include` resolve relative to the **current working directory**, not the template's own directory — matters if the Caddyfile template `{% include %}`s per-workload route fragments. [Files operations](https://docs.pyinfra.com/en/3.x/operations/files.html)

```python
# deploy_caddy.py
from pyinfra.operations import files

WORKLOADS = [
    {"name": "immich", "upstream": "127.0.0.1:2283", "path": "/immich*"},
    {"name": "ai-agents", "upstream": "127.0.0.1:8080", "path": "/agents*"},
]

files.template(
    name="Template the full Caddyfile",
    src="templates/Caddyfile.j2",
    dest="/etc/caddy/Caddyfile",
    workloads=WORKLOADS,
    control_ui_root="/srv/control-ui/dist",
    control_server_api_upstream="server.tailnet:8000",
)
```
Idempotency/diffing mechanism for `files.template` isn't spelled out in prose on that page; pyinfra's general model (confirmed by live testing below) is: render the template during prepare, diff against the current remote file content, and only write if different — a second run with unchanged inputs reports "No Change".

## Reusable multi-operation units: `@deploy` (ticket 04)

```python
from pyinfra.api import deploy
from pyinfra.operations import git, systemd

@deploy("Git-deployed systemd service")
def git_systemd_service(
    repo_url: str,
    ref: str,
    dest: str,
    unit_name: str,
    unit_src: str,          # path to a systemd unit template on the control machine
):
    repo = git.repo(
        name=f"Pull {repo_url}@{ref} to {dest}",
        src=repo_url,
        dest=dest,
        branch=ref,
        pull=True,
    )

    unit = files.template(
        name=f"Install unit file for {unit_name}",
        src=unit_src,
        dest=f"/etc/systemd/system/{unit_name}",
    )

    systemd.service(
        name=f"Enable/restart {unit_name}",
        service=unit_name,
        running=True,
        enabled=True,
        restarted=repo.did_change or unit.did_change,
        daemon_reload=unit.did_change,
    )
```
Consumed from `deploy_control_pi_api.py` / `deploy_control_server_api.py`:
```python
from control_api_deploy import git_systemd_service

git_systemd_service(
    repo_url="git@github.com:you/homelab.git",
    ref="main",
    dest="/srv/control-pi-api",
    unit_name="control-pi-api.service",
    unit_src="templates/control-pi-api.service.j2",
)
```
- `@deploy(name, data_defaults=None)` wraps a function so operations called inside it inherit deploy-wide global arguments (e.g. `_sudo`); callers can pass those globals when invoking the function (`git_systemd_service(..., sudo=True)`). [Writing Deploys](https://docs.pyinfra.com/en/3.x/api/deploys.html)
- Confirmed import path from the installed package: `from pyinfra.api import deploy`.
- `data_defaults` lets a deploy function declare overridable defaults pulled from `host.data`, e.g. `DEFAULTS = {"mariadb_version": "1.2.3"}` then `@deploy("...", data_defaults=DEFAULTS)`. [Writing Deploys](https://docs.pyinfra.com/en/3.x/api/deploys.html)
- Every operation call returns an `OperationMeta`-like handle; check `.did_change` (or the `will_change` flag surfaced in reporting) to make "restart only if something changed" logic instead of unconditionally restarting on every run — this is how ticket 04's "restarts on change, leaves alone otherwise" requirement is satisfied. [Deploy Process](https://docs.pyinfra.com/en/3.x/deploy-process.html)

An alternate, lower-ceremony composition mechanism also exists — `local.include("file.py", data={...})` — but it's file-inclusion, not a first-class parameterized function, and doesn't compose as cleanly into multiple call sites. Prefer `@deploy` for ticket 04. [Using Operations](https://docs.pyinfra.com/en/3.x/using-operations.html)

## CLI invocation

```
pyinfra [OPTIONS] INVENTORY OPERATIONS...
```
Confirmed verbatim from `pyinfra --help` (installed 3.9.2) and [Using the CLI](https://docs.pyinfra.com/en/3.x/cli.html):

```bash
# Full deploy, both groups, everything
pyinfra pyinfra/inventory.py pyinfra/deploy.py

# One Concern against one Host group
pyinfra pyinfra/inventory.py pyinfra/deploy_caddy.py --limit pi

# Same, dry-run
pyinfra pyinfra/inventory.py pyinfra/deploy_caddy.py --limit pi --dry

# Ad hoc single operation, no deploy file
pyinfra pyinfra/inventory.py server.user pyinfra home=/home/pyinfra

# Ad hoc fact gather (read-only, e.g. to sanity-check group connectivity per ticket 01)
pyinfra pyinfra/inventory.py fact server.Os --limit test

# Inspect resolved inventory/groups/data without connecting for real work
pyinfra pyinfra/inventory.py debug-inventory
```

Other relevant flags (verbatim from `pyinfra --help`): `--limit TEXT` (restrict by host or group name), `--data KEY=VALUE` / `--group-data PATH` (override data), `-y/--yes` (apply without prompting), `--diff` (show file/template diffs), `--json` (machine-readable output for facts/dry runs/results).

### `--dry`: what it actually does

Live-verified (installed 3.9.2, `@local` connector, a `files.file` operation):
- `--dry` **does connect to the host and run the prepare phase**, including fact-gathering — it is not a fully offline static check. It prints a "Detected changes" table of operations that *would* run, then disconnects without executing them.
- Nothing is mutated: no operation's execute phase runs.
- Real output observed:
  ```
  --> Detected changes:
      Operation                 Change       Conditional Change
      Ensure demo file exists   1 (@local)   -
  ```
  and when nothing would change:
  ```
  --> Detected changes:
      Operation                 Change   Conditional Change
      Ensure demo file exists   -        -
  ```
- The docs' own caveat: "Detected changes may not include every change pyinfra will execute. Hidden side effects of operations may alter behaviour of future operations." I.e. `--dry`'s preview is a **prepare-phase estimate**, not a guarantee — some operations only know their true effect once actually run. Treat ticket 08/09's `--dry` tier as "catches obvious template/logic errors," not as a complete correctness proof (that's what the disposable-container and idempotency tiers are for). [Deploy Process](https://docs.pyinfra.com/en/3.x/deploy-process.html)

## Connectors

- **`@ssh`** — the default; a bare hostname or `(hostname, {ssh_user, ssh_port, ssh_key, ssh_password, ...})` tuple in `inventory.py` uses it implicitly. This is what `pi` and `server` groups should use. [SSH connector](https://docs.pyinfra.com/en/3.x/connectors/ssh.html)
- **`@local`** — runs against the machine running pyinfra itself; useful for ticket 05's "build the Control UI on the dev machine" step (`local.shell(...)` / a `@local`-targeted host in a separate small inventory, or just plain `subprocess`/`local.shell` outside the SSH-targeted deploy). [Connectors](https://docs.pyinfra.com/en/3.x/connectors.html)
- **`@docker`** — see below (ticket 09, high priority).
- **`@dockerssh`** — Docker containers *on a remote host*, reached over SSH to the Docker host first (`@dockerssh/remotehost:image`). Documented as **beta**. Not needed here unless the disposable containers for `test` live on a remote Docker host rather than the dev machine. [Connectors](https://docs.pyinfra.com/en/3.x/connectors.html), confirmed via installed package docstring (`pyinfra/connectors/dockerssh.py`).
- **`@terraform`**, **`@vagrant`**, **`@podman`/`@podmanssh`**, **`@chroot`** — not relevant to this repo.

### `@docker` connector — ticket 09's core dependency (verified, with a load-bearing caveat)

**Yes, pyinfra 3.x has an official, documented `@docker` connector for targeting Docker containers as inventory hosts.** [Connectors](https://docs.pyinfra.com/en/3.x/connectors.html), and confirmed by reading the installed package (`pyinfra/connectors/docker.py`) since the specific `connectors/docker.html` subpage returned incomplete detail via automated fetch.

Two addressing modes, both via `@docker/<identifier>` in inventory host strings:

```python
test = [
    "@docker/debian:bookworm",       # image mode
    # or:
    "@docker/2beb8c15a1b1",          # existing-container mode (container ID/name)
]
```

- **Image mode** (`@docker/<image>`): on connect, pyinfra runs (source-verified) the equivalent of `docker run -d <image> tail -f /dev/null`, then executes operations against that container via `docker exec`. On disconnect it **commits the container to a new image, then removes the container** (`docker commit` + `docker rm -f`) — the container is disposable by construction, matching "disposable containers" in the spec's Testing Decisions almost exactly. Every invocation of `@docker/<image>` starts a **fresh** container from the pristine image — confirmed both in source and via a web search of the docs corpus, which states plainly that image-mode "will use a new container each run (meaning there will always be changes)."
- **Existing-container mode** (`@docker/<container-id-or-name>`): pyinfra execs into an already-running container and **leaves it running** afterward — state persists across runs. This is the mode that supports a genuine two-runs-in-a-row idempotency check without extra plumbing.

**The systemd caveat (flag clearly, matters for tickets 04/05/06/09):** image-mode containers are started with `tail -f /dev/null` as PID 1 (source-verified) — there is no init system running, so `systemd.service`/`systemctl` operations from the shared git+systemd helper (ticket 04) **will not work** against a freshly-spun-up `@docker/<image>` container out of the box. This is not stated as a warning anywhere in the docs prose; it's a direct consequence of how the connector starts containers, read from source. Existing-container mode sidesteps this *only if* the container was started separately (outside pyinfra) from a systemd-capable base image with the privileges systemd needs inside Docker (e.g. `--privileged`, `--cgroupns=host`, `/sys/fs/cgroup` mounted) — a well-established general Docker/Ansible-Molecule pattern, not something pyinfra documents or automates itself.

**How `--dry`/idempotency interact with image mode**: because image mode always starts from the pristine base image, running the *same* `@docker/<image>` inventory entry twice does **not** test idempotency — the second run starts fresh again. To actually exercise idempotency against `@docker`, either (a) target existing-container mode against one long-lived container across both runs, or (b) capture the image ID that image-mode prints on disconnect after run 1 and point a second inventory host at `@docker/<that-image-id>` for run 2. Neither of these two-run recipes is spelled out step-by-step in the docs — see Open Questions.

## Secrets / env vars (ticket 02: `TAILSCALE_AUTH_KEY`)

`inventory.py` and deploy files are plain Python, so the baseline pattern the docs themselves use is direct `os.environ[...]` access:
```python
import os
from pyinfra import host
from pyinfra.operations import server

auth_key = os.environ["TAILSCALE_AUTH_KEY"]

server.shell(
    name="Join tailnet",
    commands=[f"tailscale up --authkey={auth_key} --ssh"],
)
```
This mirrors the documented pattern on the [Using Secrets](https://docs.pyinfra.com/en/3.x/examples/secret_storage.html) page (which layers optional encryption-at-rest via `privy` on top of the same `os.environ[...]` read — out of scope per this repo's spec, which explicitly defers encrypted secret stores).

Separately, `config.INHERIT_ENV` (a list assigned in an optional `config.py`) forwards **named environment variables from the machine running pyinfra into the remote command environment** for every operation — useful for tools that read env vars on the *target* rather than taking a value as an operation argument. [Inventory & Data](https://docs.pyinfra.com/en/3.x/inventory-data.html) Example: `config.INHERIT_ENV = ["SOPS_AGE_KEY_FILE", "AWS_PROFILE"]`. For `TAILSCALE_AUTH_KEY`, the direct `os.environ[...]` → operation-argument pattern above is simpler and keeps the key out of the target's persistent environment, so prefer it unless a future secret genuinely needs to be read by a remote process itself.

Do **not** write secrets into `host.data`/`group_data/*.py` — those are plain committed-or-committable Python files.

## Idempotency mechanics (ticket 09, tier 3)

Live-verified end to end (installed 3.9.2, `@local`, `files.file`):

1. First real run (`-y`, no `--dry`) — operation executes, prints `Success`, and the summary table shows:
   ```
   --> Results:
       Operation                 Hosts   Success   Error   No Change
       Ensure demo file exists   1       1         -       -
       Grand total               1       1         -       -
   ```
2. Second real run against the same host, no state changed in between — prints `No changes` per-host and the summary table shows:
   ```
   --> Results:
       Operation                 Hosts   Success   Error   No Change
       Ensure demo file exists   1       -         -       1
       Grand total               1       -         -       1
   ```

**"Zero pending operations" (per ticket 09) = the `Success` column of the `Grand total` row is empty/`-` and everything lands in `No Change`.** Mechanically: every operation's Python re-diffs desired vs. current state (via facts) each time it's evaluated; if the diff produces no commands, pyinfra reports "No Change" for that host/operation and executes nothing. There's no separate "idempotency mode" — it's the same execution path every time, and "no change" is simply the observed outcome when the declared state already matches reality. [Deploy Process](https://docs.pyinfra.com/en/3.x/deploy-process.html)

For a scripted pass/fail gate, `--json` output on a normal (non-dry) run includes machine-readable per-operation change counts (confirmed by `--help`: "`--json` Emit pure JSON output on stdout (for facts, debug-inventory, debug-operations, dry runs and deploy results)"), which ticket 09's documented procedure can parse instead of eyeballing the table.

## systemd operations

Module: `pyinfra.operations.systemd`. Signature (installed 3.9.2, matches [Systemd operations](https://docs.pyinfra.com/en/3.x/operations/systemd.html)):
```python
systemd.service(
    service: str,
    running=True,
    restarted=False,
    reloaded=False,
    command: str | None = None,
    enabled: bool | None = None,
    daemon_reload=False,
    user_mode=False,
    machine: str | None = None,
    user_name: str | None = None,
)
```
```python
from pyinfra.operations import systemd

systemd.service(
    name="Enable and restart the control-pi-api service",
    service="control-pi-api.service",
    running=True,
    restarted=True,   # set conditionally via a previous op's .did_change, not always True
    enabled=True,
    daemon_reload=True,  # needed the run a new/changed unit file is installed
)
```
`systemd.daemon_reload(user_mode=False, machine=None, user_name=None)` also exists standalone if you'd rather sequence it explicitly rather than via the `daemon_reload=` kwarg on `service()`. [Systemd operations](https://docs.pyinfra.com/en/3.x/operations/systemd.html)

## git operations

Module: `pyinfra.operations.git`, operation `git.repo`. Signature (installed 3.9.2, matches [Git operations](https://docs.pyinfra.com/en/3.x/operations/git.html)):
```python
git.repo(
    src: str,
    dest: str,
    branch: str | None = None,
    pull: bool = True,
    rebase: bool = False,
    user: str | None = None,
    group: str | None = None,
    ssh_keyscan: bool = False,
    update_submodules: bool = False,
    recursive_submodules: bool = False,
    depth: int | None = None,
    *, fetch_tags: bool = False,
)
```
- Clones to `dest` if not present; if present and `pull=True` (default), fetches and pulls `branch`.
- `branch` accepts any ref `git checkout` accepts in practice (branch name, tag, or commit SHA) — the docs/signature only call it "branch to pull/checkout"; there is no separate `ref=`/`rev=`/`commit=` argument. For pinning to an exact commit, pass the SHA as `branch`.
- `ssh_keyscan=True` is relevant if the target device doesn't already trust the git remote's host key (likely true for a freshly-provisioned `pi`/`server`).

```python
git.repo(
    name="Pull control-pi-api source",
    src="git@github.com:you/homelab.git",
    dest="/srv/control-pi-api",
    branch="main",       # or a pinned commit SHA
    ssh_keyscan=True,
)
```

## Gotchas / version-specific behavior (3.x vs. stale 1.x/2.x knowledge)

- **No `--check` flag.** The spec/tickets say "`--check` (dry-run)" throughout — pyinfra 3.x's actual flag is **`--dry`**. Verified against `pyinfra --help` (3.9.2) and [Using the CLI](https://docs.pyinfra.com/en/3.x/cli.html); no `--check` string appears anywhere in the CLI reference. Implementing agents should read every "`--check`" in the tickets as "`--dry`".
- **`@deploy` decorator import path is `pyinfra.api.deploy`**, not a bare top-level import, and not (per very old material) a class-based "Deploy" object. [Writing Deploys](https://docs.pyinfra.com/en/3.x/api/deploys.html)
- **Two-phase prepare/execute model with `_if` for execute-time conditionals** — Python `if` on a previous operation's result inside a deploy file only sees prepare-time state, not actual execute-time outcomes; use the `_if` global argument when you need branching on "did the previous op actually change something." [Deploy Process](https://docs.pyinfra.com/en/3.x/deploy-process.html)
- **`server.Os` fact is deprecated** in favor of `server.Kernel` (confirmed via installed package docstring: "This fact is deprecated/renamed, please use the `server.Kernel` fact."). Prefer `server.Kernel` or `server.LinuxDistribution` for ticket 01's connectivity sanity check rather than `Os`.
- **Global arguments are prefixed with `_`** (`_sudo`, `_env`, `_if`, `_chdir`, etc.) to avoid clashing with operation-specific kwargs; only `name` is unprefixed. [Arguments](https://docs.pyinfra.com/en/3.x/arguments.html)
- **`apt-key` is deprecated upstream**; pyinfra's `apt.key` operation documents itself as using "modern keyring approaches rather than the deprecated apt-key tool" — relevant if Docker's own apt repo setup (ticket 03) or any workload needs a signing key added. [Apt operations](https://docs.pyinfra.com/en/3.x/operations/apt.html)
- **No built-in "install Docker engine" operation.** pyinfra ships generic `apt`/`files`/`server`/`systemd` operations, not a `docker.install`-style high-level operation — ticket 03 has to compose Docker's official apt-repo + `apt.packages` + `systemd.service` steps manually (or shell out to a vendor install script via `server.shell`), there's no shortcut documented.
- **Image-mode `@docker` containers are single-use per run** (fresh container from the image every time, committed-and-destroyed on disconnect) — don't assume state persists across two `pyinfra ... @docker/<image>` invocations the way it would for `pi`/`server`. See the `@docker` section above.
- **`files.template`'s Jinja2 `include`/`extends`/`import` paths resolve relative to the current working directory**, not the template file's directory — a easy-to-get-wrong detail for a multi-file Caddyfile template setup (ticket 07). [Files operations](https://docs.pyinfra.com/en/3.x/operations/files.html)

## Open questions (unresolved by docs — needs a human/future-agent decision)

1. **Exact recipe for a stable, idempotency-testable `test` Host group using `@docker`.** The docs confirm the connector exists and confirm (via source + a docs-corpus search snippet) that image-mode is single-use-per-run while existing-container-mode persists state, but **no pyinfra doc page walks through "stand up disposable containers, wire their IDs into a `test` group, run a Deploy twice, verify zero pending ops."** That workflow above is this document's synthesis of documented connector behavior, not a documented recipe. Two candidate implementations for ticket 09, neither blessed by the docs:
   - (a) Pre-start long-lived containers outside pyinfra (`docker run -d ...`), capture their container IDs (env var or a small shell wrapper), and reference `@docker/<container-id>` in `inventory.py` for the `test` group — state persists naturally across two runs.
   - (b) Chain image IDs: run 1 targets `@docker/debian:bookworm`, capture the committed image ID pyinfra prints on disconnect, run 2 targets `@docker/<that-image-id>`.
   Needs a decision before ticket 09 is implemented.
2. **systemd inside the `test` group's containers.** Confirmed (source-level) that pyinfra's own image-mode containers do not run an init system, so `systemd.service` operations from ticket 04's shared helper cannot succeed against a container spun up by `@docker/<image>` as-is. Whether the team adopts a systemd-capable base image + `--privileged`/cgroup-mount container (started outside pyinfra, referenced via existing-container mode) is unresolved and not something pyinfra documents — this is a general Docker/Ansible-Molecule technique, orthogonal to pyinfra. The spec's own fallback ("spin up containers with SSH exposed, use the normal `@ssh` connector") sidesteps the `@docker` connector's exec-based command execution entirely and would let a systemd-capable container be reached exactly like `pi`/`server` are — this may be the pragmatic choice for ticket 09 specifically because it also exercises the real `@ssh` connector path rather than `@docker`'s `docker exec` path, closer to "catch real execution errors" against the actual code path used for `pi`/`server`.
3. **`files.template`'s exact diffing/idempotency mechanism** (content hash vs. line-by-line diff) is not stated in prose on the Files operations page; behavior was inferred from pyinfra's general model and confirmed only for the simpler `files.file` operation via live testing, not `files.template` specifically. Low risk, but worth a quick live check (`files.template` against `@local` twice) before leaning on it for ticket 07's idempotency requirement.

## References

- Getting Started — https://docs.pyinfra.com/en/3.x/getting-started.html
- Using Operations — https://docs.pyinfra.com/en/3.x/using-operations.html
- Inventory & Data — https://docs.pyinfra.com/en/3.x/inventory-data.html
- Using the CLI — https://docs.pyinfra.com/en/3.x/cli.html
- Operations reference (index) — https://docs.pyinfra.com/en/3.x/operations.html
- Facts reference — https://docs.pyinfra.com/en/3.x/facts.html
- Connectors (index) — https://docs.pyinfra.com/en/3.x/connectors.html
- SSH connector — https://docs.pyinfra.com/en/3.x/connectors/ssh.html
- Docker connector — https://docs.pyinfra.com/en/3.x/connectors/docker.html
- Arguments (global) — https://docs.pyinfra.com/en/3.x/arguments.html
- Deploy Process — https://docs.pyinfra.com/en/3.x/deploy-process.html
- Writing Deploys (`@deploy`) — https://docs.pyinfra.com/en/3.x/api/deploys.html
- Files operations (`files.template`) — https://docs.pyinfra.com/en/3.x/operations/files.html
- Systemd operations — https://docs.pyinfra.com/en/3.x/operations/systemd.html
- Git operations — https://docs.pyinfra.com/en/3.x/operations/git.html
- Apt operations — https://docs.pyinfra.com/en/3.x/operations/apt.html
- Using Secrets example — https://docs.pyinfra.com/en/3.x/examples/secret_storage.html
- FAQ — https://docs.pyinfra.com/en/3.x/faq.html

Facts marked "installed package"/"source-verified" above were cross-checked against `pyinfra==3.9.2` installed from PyPI (matches the current 3.x docs line) via `pip show`, `inspect.signature`, and reading `pyinfra/connectors/docker.py`, `pyinfra/connectors/dockerssh.py`, and `pyinfra/api/host.py` directly, plus live CLI runs (`pyinfra --help`, `--dry`, and repeated real runs against `@local`) — used where the automated docs-page fetch returned incomplete prose (notably `connectors/docker.html`) or where direct execution was the more authoritative source than describing it secondhand.
