"""Host groups for the homelab control plane.

Two real devices, `gateway` and `compute`, plus a `test` group of disposable,
systemd-capable containers standing in for both devices' OS bases. No
Deploy file hardcodes a host -- they target `gateway`/`compute`/`test` by name.

Real addresses are read from the environment via `settings.InventorySettings`
(pydantic-settings), not hardcoded, with a placeholder MagicDNS-shaped
default.

`device_role` (host data, not a pyinfra group) records which application
logic a host runs -- "gateway" or "compute" -- independently of whether it's
the real device or a disposable stand-in. Test hosts aren't added into the
`gateway`/`compute` groups themselves, since those groups must stay
targetable in isolation via `--limit`, which a shared group name would
break. See docs/agents/pyinfra.md for why `test` uses the `@ssh` connector
rather than `@docker` image mode. pyinfra/README.md documents how to stand
up these containers.
"""

from settings import InventorySettings

settings = InventorySettings()

gateway = [
    (settings.gateway_host, {"ssh_user": settings.gateway_ssh_user, "device_role": "gateway"}),
]

compute = [
    (
        settings.compute_host,
        {
            "ssh_user": settings.compute_ssh_user,
            "device_role": "compute",
            "_sudo_password": settings.compute_sudo_password,
        },
    ),
]

# Separate inventory identities ("gateway-test"/"compute-test") with the
# connect address in `ssh_hostname` -- pyinfra dedupes hosts by identity
# string, and both containers share the same dev-machine address on
# different ports.
test = [
    (
        "gateway-test",
        {
            "ssh_hostname": settings.gateway_test_host,
            "ssh_port": settings.gateway_test_ssh_port,
            "ssh_user": settings.gateway_test_ssh_user,
            "device_role": "gateway",
        },
    ),
    (
        "compute-test",
        {
            "ssh_hostname": settings.compute_test_host,
            "ssh_port": settings.compute_test_ssh_port,
            "ssh_user": settings.compute_test_ssh_user,
            "device_role": "compute",
        },
    ),
]
