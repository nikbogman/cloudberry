# Deploy

[Task](https://taskfile.dev/) over plain `ssh`, defined in the repo root
[`Taskfile.yml`](Taskfile.yml). One run converges raspberry and blackberry:
Docker installed on blackberry, and waker and hostd deployed as systemd services
served on the tailnet.

Per-service deploy details:
[waker](cmd/waker/README.md),
[hostd](cmd/hostd/README.md). Vocabulary is in
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
[waker](cmd/waker/README.md#environment) and
[hostd](cmd/hostd/README.md#environment).

## Running a Deploy

```sh
task                # Docker + waker + hostd
task waker          # just waker, on raspberry
task hostd          # just hostd, on blackberry
task docker         # just Docker on blackberry
task tailscale      # install + join the tailnet, both devices (not part of `task`)
task --dry          # print the commands without running them
task --list
```

Every run re-ships and restarts each service it targets: there is no change
detection. A restart is a second of downtime, which nothing here cares
about. Nothing runs a Deploy automatically.

### Renaming a service

A Deploy only knows the current binary names, so the old unit keeps running
after a rename. Remove the old one first — it still holds the port — then
deploy the new name:

```sh
task cleanup DEVICE=RASPBERRY OLD=old-binary-name
task waker
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
RASPBERRY_HOST=waker-test RASPBERRY_SSH_USER=root task waker
```

`tailscale serve` fails in a container that isn't on the tailnet; everything
before it is exercised.

Bind-address enforcement and the system's other limitations are in
[`DESIGN.md`](DESIGN.md#known-limitations).
