#!/usr/bin/env bash
# Wrapper that loads secrets.sh (ADR-0010, gitignored, never
# committed) into THIS SCRIPT'S OWN PROCESS ONLY, then execs pyinfra. The
# vars never touch the interactive shell that invoked this script -- once
# the process exits, nothing lingers. Also bakes in inventory.py, and
# resolves a short name (e.g. "caddy") to its deploy_caddy.py file, so
# callers don't have to keep retyping every Deploy file's "deploy" prefix.
#
# Usage: ./deploy.sh --limit pi --dry           # no target -> deploy.py (everything)
#        ./deploy.sh caddy --limit pi --dry     # -> deploy_caddy.py
#        ./deploy.sh deploy.py --limit pi       # full filename still works
#        ./deploy.sh --version                  # flags pass through untouched
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

if [[ $# -eq 0 || "$1" == -* ]]; then
  resolved="deploy.py"
else
  target="$1"
  shift
  if [[ -f "$target" ]]; then
    resolved="$target"
  elif [[ -f "deploy_${target}.py" ]]; then
    resolved="deploy_${target}.py"
  else
    echo "deploy.sh: no Deploy file matches '$target' (tried '$target', 'deploy_${target}.py')" >&2
    echo "available: $(ls deploy*.py | tr '\n' ' ')" >&2
    exit 1
  fi
fi
set -- "$resolved" "$@"

set -a
# shellcheck source=secrets.sh
source secrets.sh
set +a

exec uv run pyinfra inventory.py "$@"
