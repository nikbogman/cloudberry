# Manual-only deploy trigger

A Deploy only ever runs when explicitly invoked from the dev machine — there is no CI pipeline, cron job, or other automatic trigger that re-applies the pyinfra configuration. We rejected scheduled drift-correction or on-push auto-deploy because an unattended Deploy hitting the homelab while unsupervised is a risk not worth taking for a two-device personal setup, particularly since some Concerns (Docker, Caddy) can disrupt actively-running workloads if applied at the wrong moment. Manual triggering puts that timing decision in the operator's hands.
