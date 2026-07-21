"""Host groups for the homelab control plane (ticket 01).

Two real devices, `pi` and `server` (CONTEXT.md's Host group entry), plus a
`test` group of disposable, systemd-capable containers standing in for
both devices' OS bases (ticket 09) — no Deploy file ever hardcodes a host;
they only ever target `pi`/`server`/`test` by name.

Real addresses are operator-specific and not invented here: they're read
from the environment, with a placeholder MagicDNS-shaped default, the same
discipline `services/auto_wake_proxy/Caddyfile` uses for the main server's
address.

`device_kind` (host data, not a pyinfra group -- deliberately not named
"role", which CONTEXT.md's Host group entry explicitly reserves against:
`_Avoid_: role (in the Ansible sense)`) records which application logic a
host runs — `"pi"` or `"server"` — independently of whether it's the real
device or a disposable stand-in. This is deliberately *not* done by adding
test hosts into the `pi`/`server` groups themselves: ticket 02 requires
`pi`/`server` to be targetable "in isolation" via `--limit`, which a test
container sharing that group name would silently break. See
docs/agents/pyinfra.md's Open Question #2 for why `test` uses the `@ssh`
connector against long-lived containers rather than `@docker` image mode:
image-mode containers run no init system, so the `systemd.service`
operations used throughout this feature (tickets 02-06) can't succeed
against them. pyinfra/README.md documents how to stand up these
containers.
"""

import os

pi = [
    (
        os.environ.get("PI_HOST", "pi-zero.tailnet"),
        {"ssh_user": os.environ.get("PI_SSH_USER", "pi"), "device_kind": "pi"},
    ),
]

server = [
    (
        os.environ.get("SERVER_HOST", "main-server.tailnet"),
        {"ssh_user": os.environ.get("SERVER_SSH_USER", "admin"), "device_kind": "server"},
    ),
]

# Both stand-in containers commonly run on the same dev-machine Docker
# daemon, reachable at the same address on different ports -- so each gets
# its own inventory identity ("pi-test"/"server-test") with the actual
# connect address in `ssh_hostname`, rather than using that address as the
# identity itself (two hosts sharing one identity string would collapse
# into a single deduplicated inventory entry).
test = [
    (
        "pi-test",
        {
            "ssh_hostname": os.environ.get("PI_TEST_HOST", "localhost"),
            "ssh_port": int(os.environ.get("PI_TEST_SSH_PORT", "2201")),
            "ssh_user": os.environ.get("PI_TEST_SSH_USER", "root"),
            "device_kind": "pi",
        },
    ),
    (
        "server-test",
        {
            "ssh_hostname": os.environ.get("SERVER_TEST_HOST", "localhost"),
            "ssh_port": int(os.environ.get("SERVER_TEST_SSH_PORT", "2202")),
            "ssh_user": os.environ.get("SERVER_TEST_SSH_USER", "root"),
            "device_kind": "server",
        },
    ),
]
