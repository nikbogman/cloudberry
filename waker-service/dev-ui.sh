#!/usr/bin/env bash
# Fast-iteration UI dev server: serves ui/ straight from disk, so edits
# show up on refresh with no rebuild. Writes the gitignored ui/config.js
# that the Waker would otherwise serve at /config.js, deriving it from
# the repo root .env the same way a Deploy does (SLEEPER_TAILNET_HOST is
# a homelab-wide device address, not one of this service's own vars).
#
# Usage: ./dev-ui.sh [port]        # default 5173
#
# Wake does NOT work here -- only the Waker serves it. Run
# `go run ./cmd/waker-api` for the real thing; it serves the *embedded*
# UI, so edits need a restart. See docs/ui.md.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

port="${1:-5173}"

# Loaded into this script's own process, never the interactive shell --
# same trick as deploy/deploy.sh. Absent .env just means no Sleeper host.
if [[ -f ../.env ]]; then
  set -a
  # shellcheck source=../.env
  source ../.env
  set +a
fi

sleeper_host="${SLEEPER_TAILNET_HOST:-}"
if [[ -n "$sleeper_host" ]]; then
  printf 'window.SLEEPER_API_URL = "https://%s"\n' "$sleeper_host" >ui/config.js
  echo "dev-ui: Sleeper API -> https://$sleeper_host"
else
  printf 'window.SLEEPER_API_URL = ""\n' >ui/config.js
  echo "dev-ui: no SLEEPER_TAILNET_HOST in ../.env -- Sleeper API is same-origin," \
    "so the status will sit at Unreachable"
fi

echo "dev-ui: http://localhost:$port"
exec python3 -m http.server "$port" --bind 127.0.0.1 --directory ui
