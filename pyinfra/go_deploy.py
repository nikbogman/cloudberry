"""Shared off-device Go build + systemd deploy helper.

The pattern common to both the Pi API and Server API now that they're Go
binaries: cross-compile locally, ship only the compiled binary, and
install/enable a systemd unit that runs it directly -- the same
off-device build-then-ship shape ADR-0013 established for the
wake_plugin-enabled Caddy binary, extended here to both control-plane
services (ADR-0015). Neither `pi` nor `server` ever runs a Go toolchain,
same reasoning as ADR-0013: the Pi Zero W is too weak to build on, and the
main server has no reason to carry a build toolchain it'll only ever use
for these two binaries.

This supersedes `api_deploy.py`'s git-pull + `uv sync` model for these two
apps: there's no source tree to check out or dependencies to install on
the target device, so `unit_name`'s ExecStart is just the shipped binary's
path.

Restart/reload gating uses each operation's `.will_change` -- the
prepare-time diff result, safe to read immediately (unlike `.did_change()`,
which raises until the operation has actually executed).

Note on `--dry`: the local `go build` always runs, even under `--dry` --
pyinfra's dry-run guarantee only covers remote operations, not a local
build step (same caveat `deploy_caddy.py` and `deploy_pi_api.py`'s UI
build already document).
"""

from pyinfra import local
from pyinfra.operations import files, systemd


def go_binary_systemd_service(
    *,
    module_dir: str,
    package: str,
    goos: str,
    goarch: str,
    binary_name: str,
    remote_binary: str,
    unit_name: str,
    description: str,
    goarm: str | None = None,
    environment: dict[str, str] | None = None,
    run_as_user: str = "root",
):
    """Cross-compile `package` (an import path relative to `module_dir`,
    e.g. "./cmd/pi-api") for `goos`/`goarch`(/`goarm`) on the dev machine,
    ship the resulting binary to `remote_binary`, install/enable a systemd
    unit named `unit_name` running it directly, and restart iff this
    Deploy changed the binary or the unit file.
    """

    local_binary = f"{module_dir}/dist/{binary_name}"
    goenv = f"GOOS={goos} GOARCH={goarch}"
    if goarm:
        goenv += f" GOARM={goarm}"

    local.shell(
        f"mkdir -p {module_dir}/dist && "
        f"cd {module_dir} && {goenv} go build -o dist/{binary_name} {package}",
        print_output=True,
    )

    binary = files.put(
        name=f"Ship the {binary_name} binary",
        src=local_binary,
        dest=remote_binary,
        mode="755",
    )

    unit = files.template(
        name=f"Install unit file for {unit_name}",
        src="templates/binary.service.j2",
        dest=f"/etc/systemd/system/{unit_name}",
        description=description,
        binary_path=remote_binary,
        environment=environment or {},
        run_as_user=run_as_user,
    )

    systemd.service(
        name=f"Enable/restart {unit_name}",
        service=unit_name,
        running=True,
        enabled=True,
        restarted=binary.will_change or unit.will_change,
        daemon_reload=unit.will_change,
    )
