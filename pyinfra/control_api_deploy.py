"""Ticket 04: shared git-pull + systemd deploy helper.

The pattern common to both Control APIs (repo layout decision in
`.scratch/pyinfra-provisioning/spec.md`): pull this repo to a target
device checked out to a specific ref (ADR-0006), install/enable a systemd
unit (ADR-0007), and restart the service only when the pulled code or the
unit file actually changed -- never unconditionally on every Deploy.

Parameterized (ref, unit name, working directory, start command) rather
than hardcoded to one app, so `deploy_control_pi_api.py` (ticket 05) and
`deploy_control_server_api.py` (ticket 06) can each consume it despite
being different applications on different Host groups. Demoed for real
against a throwaway app/unit -- see `docs/agents/pyinfra-demo.md` -- before
either real Control API Deploy file adopted it.

Restart/reload gating uses each operation's `.will_change` -- the
prepare-time diff result, safe to read immediately after calling the
operation (unlike `.did_change()`, which raises until the operation has
actually executed; see docs/agents/pyinfra.md's Deploy Process notes and
the pyinfra `OperationMeta` source, `pyinfra/api/operation.py`).
"""

from pyinfra.api import deploy
from pyinfra.operations import files, git, server, systemd


@deploy("Git-deployed systemd service")
def git_systemd_service(
    *,
    repo_url: str,
    ref: str,
    dest: str,
    unit_name: str,
    description: str,
    start_command: str,
    working_directory: str | None = None,
    environment: dict[str, str] | None = None,
    setup_commands: list[str] | None = None,
    run_as_user: str = "root",
):
    """Pull `repo_url`@`ref` to `dest`, install/enable a systemd unit named
    `unit_name` running `start_command`, and restart it iff this Deploy
    changed the pulled code or the unit file.

    `working_directory` (the unit's `WorkingDirectory=` and where
    `setup_commands` run) defaults to `dest` but can be overridden to a
    subdirectory of it -- this repo is a monorepo, so `dest` is a full
    clone while each app actually lives under `dest/services/<app>`.

    `setup_commands` (e.g. `uv sync`) run only when the pull actually
    changed something -- there's no point reinstalling dependencies on a
    Deploy that pulled nothing new.
    """

    working_directory = working_directory or dest

    repo = git.repo(
        name=f"Pull {repo_url}@{ref} to {dest}",
        src=repo_url,
        dest=dest,
        branch=ref,
        ssh_keyscan=True,
    )
    code_changed = repo.will_change

    if setup_commands and code_changed:
        server.shell(
            name=f"Install dependencies for {unit_name}",
            commands=setup_commands,
            _chdir=working_directory,
        )

    unit = files.template(
        name=f"Install unit file for {unit_name}",
        src="templates/control-api.service.j2",
        dest=f"/etc/systemd/system/{unit_name}",
        description=description,
        working_directory=working_directory,
        start_command=start_command,
        environment=environment or {},
        run_as_user=run_as_user,
    )

    systemd.service(
        name=f"Enable/restart {unit_name}",
        service=unit_name,
        running=True,
        enabled=True,
        restarted=code_changed or unit.will_change,
        daemon_reload=unit.will_change,
    )
