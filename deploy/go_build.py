"""Shared off-device Go build + systemd deploy helper, used by both the
Gateway API and Compute API: cross-compile locally, ship only the binary,
install/enable a systemd unit that runs it directly. Neither device ever
runs a Go toolchain.

Restart/reload gating uses each operation's `.will_change` -- the
prepare-time diff, safe to read immediately (unlike `.did_change()`,
which raises until the operation has actually executed).

The local `go build` always runs, even under `--dry` -- pyinfra's dry-run
guarantee only covers remote operations.
"""

from pyinfra import local
from pyinfra.operations import files, server, systemd


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
    e.g. "./cmd/gateway") for `goos`/`goarch`(/`goarm`) on the dev machine,
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

    # Staged at a stable path (not remote_binary directly, never deleted)
    # so the diff below stays accurate across runs.
    staged_binary = f"{remote_binary}.new"
    binary = files.put(
        name=f"Ship the {binary_name} binary",
        src=local_binary,
        dest=staged_binary,
        mode="755",
    )

    if binary.will_change:
        # Truncating a running binary in place hits ETXTBSY; `mv` instead
        # only swaps the directory entry, which the kernel always allows.
        # staged_binary is left intact for the next run's diff.
        swap_tmp = f"{remote_binary}.swap"
        server.shell(
            name=f"Move the staged {binary_name} binary into place",
            commands=[f"cp {staged_binary} {swap_tmp} && mv -f {swap_tmp} {remote_binary}"],
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
