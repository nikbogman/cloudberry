"""Host groups for the homelab control plane.

Two real devices, `gateway` and `compute`, plus a `test` group of
disposable, systemd-capable containers standing in for both. No Deploy
file hardcodes a host -- they target `gateway`/`compute`/`test` by name.
Real addresses come from `settings.InventorySettings`.

`device_role` (host data, not a pyinfra group) records which application
logic a host runs, independent of whether it's the real device or its
`test` stand-in. Test hosts aren't added to the `gateway`/`compute`
groups themselves, since those must stay targetable in isolation via
`--limit`. See deploy/README.md for how to stand up these
containers.
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

# Separate identities ("gateway-test"/"compute-test") with the connect
# address in `ssh_hostname` -- pyinfra dedupes hosts by identity, and both
# containers share the same dev-machine address on different ports.
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
