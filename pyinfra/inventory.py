"""Host groups for the homelab control plane.

Two real devices, `pi` and `server`, plus a `test` group of disposable,
systemd-capable containers standing in for both devices' OS bases. No
Deploy file hardcodes a host -- they target `pi`/`server`/`test` by name.

Real addresses are read from the environment via `settings.InventorySettings`
(pydantic-settings), not hardcoded, with a placeholder MagicDNS-shaped
default.

`device_role` (host data, not a pyinfra group) records which application
logic a host runs -- "pi" or "server" -- independently of whether it's the
real device or a disposable stand-in. Test hosts aren't added into the
`pi`/`server` groups themselves, since those groups must stay targetable
in isolation via `--limit`, which a shared group name would break. See
docs/agents/pyinfra.md for why `test` uses the `@ssh` connector rather than
`@docker` image mode. pyinfra/README.md documents how to stand up these
containers.
"""

from settings import InventorySettings

settings = InventorySettings()

pi = [
    (settings.pi_host, {"ssh_user": settings.pi_ssh_user, "device_role": "pi"}),
]

server = [
    (settings.server_host, {"ssh_user": settings.server_ssh_user, "device_role": "server"}),
]

# Separate inventory identities ("pi-test"/"server-test") with the connect
# address in `ssh_hostname` -- pyinfra dedupes hosts by identity string, and
# both containers share the same dev-machine address on different ports.
test = [
    (
        "pi-test",
        {
            "ssh_hostname": settings.pi_test_host,
            "ssh_port": settings.pi_test_ssh_port,
            "ssh_user": settings.pi_test_ssh_user,
            "device_role": "pi",
        },
    ),
    (
        "server-test",
        {
            "ssh_hostname": settings.server_test_host,
            "ssh_port": settings.server_test_ssh_port,
            "ssh_user": settings.server_test_ssh_user,
            "device_role": "server",
        },
    ),
]
