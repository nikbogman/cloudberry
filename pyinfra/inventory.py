"""Host groups for the homelab control plane.

Two real devices, `pi` and `server`, plus a `test` group of disposable,
systemd-capable containers standing in for both devices' OS bases. No
Deploy file hardcodes a host -- they target `pi`/`server`/`test` by name.

Real addresses are read from the environment, not hardcoded, with a
placeholder MagicDNS-shaped default.

`device_kind` (host data, not a pyinfra group) records which application
logic a host runs -- "pi" or "server" -- independently of whether it's the
real device or a disposable stand-in. Test hosts aren't added into the
`pi`/`server` groups themselves, since those groups must stay targetable
in isolation via `--limit`, which a shared group name would break. See
docs/agents/pyinfra.md for why `test` uses the `@ssh` connector rather than
`@docker` image mode. pyinfra/README.md documents how to stand up these
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

# Separate inventory identities ("pi-test"/"server-test") with the connect
# address in `ssh_hostname` -- pyinfra dedupes hosts by identity string, and
# both containers share the same dev-machine address on different ports.
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
