# Suspend-to-RAM only, no full shutdown

The Server API and UI support only suspend-to-RAM as a sleep state; there is no full power-off (ACPI S5) action anywhere in the system, including the future autosuspend feature. We rejected supporting full shutdown because Wake-on-LAN after a complete power-off is unreliable across BIOS and NIC firmware configurations, while resuming from suspend is fast and predictable. A future reader adding a "shut down" button should know this was a deliberate omission, not an oversight.
