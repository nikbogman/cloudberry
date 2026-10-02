# Deploy

[Task](https://taskfile.dev/) over plain `ssh`, defined in the repo root
[`Taskfile.yml`](../Taskfile.yml). One run converges the Waker and the Sleeper:
Docker installed on the Sleeper, and both binaries deployed as systemd services
served on the tailnet.

Per-service deploy details:
[`waker.md`](waker.md),
[`sleeper.md`](sleeper.md). Vocabulary is in
[`CONTEXT.md`](../CONTEXT.md).

## Setup

Needs `task` (`go install github.com/go-task/task/v3/cmd/task@latest`), a Go
toolchain, and ssh access to both devices.

Copy `.env.example` to `.env` (gitignored) and fill it in. Task loads it via
`dotenv`, into its own process only.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `WAKER_HOST`, `WAKER_SSH_USER` | `pi-zero.tailnet`, `pi` | Waker's ssh target |
| `SLEEPER_HOST`, `SLEEPER_SSH_USER` | `main-server.tailnet`, `admin` | Sleeper's ssh target |
| `WAKER_SUDO_PASSWORD`, `SLEEPER_SUDO_PASSWORD` | — | Fed to `sudo -S`; only needed where sudo asks for one (e.g. sudo-rs on the Sleeper) |
| `TAILSCALE_AUTH_KEY` | — | Only needed for a device's first join |

`WAKER_HOST`/`SLEEPER_HOST` are ssh targets — on a fresh device that hasn't
joined the tailnet, use a LAN address. Any `~/.ssh/config` alias works too,
including one with a non-default port.

Service-specific variables are documented in
[`waker.md`](waker.md#deploy-time-variables) and
[`sleeper.md`](sleeper.md#deploy-time-variables).

## Running a Deploy

```sh
task                # Docker + Waker API + Sleeper API
task waker          # just the Waker API
task sleeper        # just the Sleeper API
task docker         # just Docker on the Sleeper
task tailscale      # install + join the tailnet, both devices (not part of `task`)
task --dry          # print the commands without running them
task --list
```

Every run re-ships both binaries and restarts both services — there is no
change detection. A restart is a second of downtime, which nothing here cares
about. Nothing runs a Deploy automatically.

### Renaming a service

A Deploy only knows the current binary names, so the old unit keeps running
after a rename. Deploy the new name, then remove the old one:

```sh
task waker
task cleanup DEVICE=WAKER OLD=old-binary-name
```

`cleanup` stops and disables `OLD`, deletes its unit and binary, and reloads
systemd. If the old service listened on a different port, also drop its
`tailscale serve` entry by hand.

## Testing

Run against systemd-capable throwaway containers by pointing the ssh targets at
them:

```sh
docker run -d --name waker-test --privileged --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw -p 2201:22 jrei/systemd-debian:12
# enable openssh-server, install sudo, seed authorized_keys via `docker exec`

# ~/.ssh/config: Host waker-test / HostName localhost / Port 2201 / User root
WAKER_HOST=waker-test WAKER_SSH_USER=root task waker
```

`tailscale serve` fails in a container that isn't on the tailnet; everything
before it is exercised.

Bind-address enforcement and the system's other limitations are in
[`ARCHITECTURE.md`](../ARCHITECTURE.md#constraints).
