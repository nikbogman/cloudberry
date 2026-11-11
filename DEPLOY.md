# Deploy

[Task](https://taskfile.dev/) over plain `ssh`, defined in the repo root
[`Taskfile.yml`](Taskfile.yml). One run converges raspberry and blackberry:
Docker installed on blackberry, the stack sync set up, and gateway and hostd
deployed as systemd services served on the tailnet. `task ui` uploads the UI
to Cloudflare Pages separately.

A Deploy is for setting a device up, and for what doesn't update itself:
unit files, environment and secrets. After that:

- **Stacks** deploy on a push to [deploys](https://github.com/nikbogman/deploys)'
  `main`: hostd pulls it ([stack sync](cmd/hostd/README.md#stack-sync)), and
  gateway picks up route changes from hostd.
- **gateway and hostd** update themselves to the latest
  [release](https://github.com/nikbogman/metal/releases), if they're a
  release build ([updates](cmd/gateway/README.md#updates)).

Per-service deploy details:
[gateway](cmd/gateway/README.md),
[hostd](cmd/hostd/README.md),
[ui](ui/README.md#deployment). Vocabulary is in
[`CONTEXT.md`](CONTEXT.md).

## Setup

Needs `task` (`go install github.com/go-task/task/v3/cmd/task@latest`), a Go
toolchain, and ssh access to both devices.

Copy `.env.example` to `.env` (gitignored) and fill it in. Task loads it via
`dotenv`, into its own process only.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `RASPBERRY_HOST`, `RASPBERRY_SSH_USER` | `raspberry`, `pi` | raspberry's ssh target |
| `BLACKBERRY_HOST`, `BLACKBERRY_SSH_USER` | `blackberry`, `admin` | blackberry's ssh target |
| `RASPBERRY_SUDO_PASSWORD`, `BLACKBERRY_SUDO_PASSWORD` | — | Fed to `sudo -S`; only needed where sudo asks for one (e.g. sudo-rs on blackberry) |
| `TAILSCALE_AUTH_KEY` | — | Only needed for a device's first join |

`RASPBERRY_HOST`/`BLACKBERRY_HOST` are ssh targets — on a fresh device that hasn't
joined the tailnet, use a LAN address. Any `~/.ssh/config` alias works too,
including one with a non-default port.

Service-specific variables are documented in
[gateway](cmd/gateway/README.md#environment) and
[hostd](cmd/hostd/README.md#environment).

## Running a Deploy

```sh
task                # Docker + HDD spindown + network + stacks + firewall + gateway + hostd
task stacks         # sops/age/keys/deploys clone on blackberry; encrypt stack secrets (see below)
task gateway        # just gateway, on raspberry (routes generated from STACKS_DIR)
task firewall       # just the gated-stack firewall, on blackberry
task network        # just blackberry's netplan config; auto-rolls back after 2 min if blackberry drops off
task hdd            # just hd-idle (HDD spindown), on blackberry
task hostd          # just hostd, on blackberry
task ui             # the UI, to Cloudflare Pages (not part of `task`)
task docker         # just Docker on blackberry
task tailscale      # install + join the tailnet, blackberry only (not part of `task`)
task --dry          # print the commands without running them
task --list
```

Every run re-ships and restarts each service it targets: there is no change
detection.

### Stack sync setup (`task stacks`)

Idempotent; rerun it after replacing either machine's age key.

- On blackberry: installs git, age and a pinned sops; generates
  `/etc/hostd/age.key` and a deploy key, `/etc/hostd/deploy_key`, if missing
  (neither private key ever leaves it); clones deploys to `/srv/deploys`.
- Registers the deploy key on deploys, read-only, if `gh` is installed and
  logged in. Otherwise it prints the key and where to add it; then rerun.
- On this machine: installs sops and age with `go install`, generates
  `~/.config/sops/age/keys.txt` if missing, writes both public keys into
  deploys' `.sops.yaml`, encrypts any stack's `.env` that has no
  `secrets.env` yet, and re-encrypts every `secrets.env` for the current keys.
  Commit and push what it changed in deploys.

Back this machine's age key up in a password manager: with it and blackberry
gone, the stack secrets are unreadable.

### Releases

Tag `vX.Y.Z` on `main` and push the tag. [CI](.github/workflows/ci.yml)
vets, tests and builds `gateway-linux-arm`, `hostd-linux-amd64` and
`SHA256SUMS` into a GitHub release; both devices pick it up within
`UPDATE_MINUTES`. A `task gateway`/`task hostd` from a clean checkout of that
tag deploys a binary that updates itself from then on; from anything else, a
binary that stays as deployed. A restart is a second of downtime, which nothing here cares
about. Nothing runs a Deploy automatically.

### Renaming a service

A Deploy only knows the current binary names, so the old unit keeps running
after a rename. Remove the old one first — it still holds the port — then
deploy the new name:

```sh
task cleanup DEVICE=RASPBERRY OLD=old-binary-name
task gateway
```

`cleanup` stops and disables `OLD`, deletes its unit and binary, and reloads
systemd. On blackberry, if the old service listened on a different port, also
drop its `tailscale serve` entry by hand.

## Testing

Run against systemd-capable throwaway containers by pointing the ssh targets at
them:

```sh
docker run -d --name gateway-test --privileged --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw -p 2201:22 jrei/systemd-debian:12
# enable openssh-server, install sudo, seed authorized_keys via `docker exec`

# ~/.ssh/config: Host gateway-test / HostName localhost / Port 2201 / User root
RASPBERRY_HOST=gateway-test RASPBERRY_SSH_USER=root task gateway
```

`tailscale serve` fails in a container that isn't on the tailnet; everything
before it is exercised.

Bind-address enforcement and the system's other limitations are in
[`DESIGN.md`](DESIGN.md#known-limitations).
