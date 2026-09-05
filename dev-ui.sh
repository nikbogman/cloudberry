#!/usr/bin/env bash
# Fast-iteration UI dev server: serves ui/ straight from disk, so edits
# show up on refresh with no rebuild. Writes the gitignored ui/config.js
# that the Gateway would otherwise serve at /config.js, deriving it from
# the repo root .env the same way a Deploy does.
#
# Usage: ./dev-ui.sh [port]        # default 5173
#
# Wake and the /server/* proxy do NOT work here -- only the Gateway
# serves those. Run `go run ./cmd/gateway-api` for the real thing; it
# serves the *embedded* UI, so edits need a restart. See docs/ui.md.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

port="${1:-5173}"

# Loaded into this script's own process, never the interactive shell --
# same trick as deploy/deploy.sh. Absent .env just means no Compute host.
if [[ -f .env ]]; then
  set -a
  # shellcheck source=.env
  source .env
  set +a
fi

compute_host="${COMPUTE_TAILNET_HOST:-}"
if [[ -n "$compute_host" ]]; then
  printf 'window.COMPUTE_API_URL = "https://%s"\n' "$compute_host" >ui/config.js
  echo "dev-ui: Compute API -> https://$compute_host"
else
  printf 'window.COMPUTE_API_URL = ""\n' >ui/config.js
  echo "dev-ui: no COMPUTE_TAILNET_HOST in .env -- Compute API is same-origin," \
    "so the status will sit at Unreachable"
fi

echo "dev-ui: http://localhost:$port"
exec python3 -m http.server "$port" --bind 127.0.0.1 --directory ui
