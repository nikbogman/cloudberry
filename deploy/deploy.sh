#!/usr/bin/env bash
# Loads the repo root's .env into this script's own process (never the
# interactive shell), execs pyinfra with inventory.py, and resolves a
# short name (e.g. "waker") to its deploy_waker.py file.
#
# Usage: ./deploy.sh --limit waker --dry           # no target -> deploy.py (everything)
#        ./deploy.sh waker --limit waker --dry   # -> deploy_waker.py
#        ./deploy.sh deploy.py --limit waker       # full filename still works
#        ./deploy.sh --version                       # flags pass through untouched
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
# shellcheck source=../.env
source ../.env
set +a

exec uv run pyinfra inventory.py "$@"
