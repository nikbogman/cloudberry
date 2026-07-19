# Secrets via dev-machine environment variables

Secrets needed during a Deploy (starting with the Tailscale auth key) are read from environment variables on the dev machine at Deploy time and passed to the relevant operation — nothing secret is ever committed to the repo. We considered an encrypted-secrets-file approach (e.g. sops or age) and rejected it as unnecessary complexity: that pattern earns its cost when a team needs to share secrets through version control, which doesn't apply to a single-operator homelab.
