# Auto-wake proxy triggers wake via the Control Pi API, not its own WoL send

The auto-wake proxy's Caddy plugin now triggers a wake by calling the Control Pi API's `POST /wake` over loopback HTTP, rather than building and broadcasting its own Wake-on-LAN packet. This replaces the third-party `github.com/dulli/caddy-wol` plugin with a small custom Go plugin (`services/auto_wake_proxy/wake_plugin/`), and collapses WoL-sending to one implementation (the Control Pi API's `wol.py`) instead of two independently-maintained ones.

This revises ADR-0005: the two wake paths are no longer independent — the auto-wake proxy now depends on the Control Pi API's availability, exactly the coupling ADR-0005 chose to avoid. We accepted that trade-off because both processes already run on the same Pi Zero and share the same physical failure domain: if the Control Pi API process is down, the Control UI's Wake button is already unusable, so the auto-wake path failing too isn't a new failure mode, just a shared one. What ADR-0005 got right and this doesn't change: the two paths remain distinct *triggers* (automatic-on-request vs. deliberate-button-press), each still driving its own caller-facing behavior (transparent hold/retry vs. the Control UI's reachability display).

Consequences:

- `/wake` now accepts a loopback-origin caller in addition to the `Tailscale-User-Login` header (ADR-0004 amended accordingly).
- The plugin throttles repeat `/wake` calls per upstream (~10 minutes, matching `caddy-wol`'s prior behavior) so a held/retried request storm doesn't hammer the Control Pi API.
- `SERVER_MAC_ADDRESS`/`WOL_BROADCAST_ADDRESS` no longer need to exist in Caddy's own environment at all — that knowledge now lives solely with the Control Pi API, where it already lived for the deliberate Wake button path.
