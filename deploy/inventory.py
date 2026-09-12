"""Host groups for the homelab.

Two real devices, `waker` and `sleeper`, plus a `test` group of
disposable, systemd-capable containers standing in for both. No Deploy
file hardcodes a host -- they target `waker`/`sleeper`/`test` by name.
Real addresses come from `settings.InventorySettings`.

`device_role` (host data, not a pyinfra group) records which application
logic a host runs, independent of whether it's the real device or its
`test` stand-in. Test hosts aren't added to the `waker`/`sleeper`
groups themselves, since those must stay targetable in isolation via
`--limit`. See docs/deploy.md for how to stand up these
containers.
"""

from settings import InventorySettings

settings = InventorySettings()

waker = [
    (
        settings.waker_host,
        {
            "ssh_user": settings.waker_ssh_user,
            "device_role": "waker",
            "_sudo": True,
        },
    ),
]

sleeper = [
    (
        settings.sleeper_host,
        {
            "ssh_user": settings.sleeper_ssh_user,
            "device_role": "sleeper",
            "_sudo": True,
            "_sudo_password": settings.sleeper_sudo_password,
        },
    ),
]

# Separate identities ("waker-test"/"sleeper-test") with the connect
# address in `ssh_hostname` -- pyinfra dedupes hosts by identity, and both
# containers share the same dev-machine address on different ports.
test = [
    (
        "waker-test",
        {
            "ssh_hostname": settings.waker_test_host,
            "ssh_port": settings.waker_test_ssh_port,
            "ssh_user": settings.waker_test_ssh_user,
            "device_role": "waker",
        },
    ),
    (
        "sleeper-test",
        {
            "ssh_hostname": settings.sleeper_test_host,
            "ssh_port": settings.sleeper_test_ssh_port,
            "ssh_user": settings.sleeper_test_ssh_user,
            "device_role": "sleeper",
        },
    ),
]
