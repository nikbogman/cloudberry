#!/usr/bin/env bash
# Loads the repo root's .env plus each service's own .env into this
# script's own process (never the interactive shell), execs pyinfra with
# inventory.py, and resolves a short name (e.g. "waker") to its Deploy
# file -- searching the top level (homelab-wide infrastructure) and then
# each service folder, so a short name works wherever the file lives.
#
# Usage: ./deploy.sh --limit waker --dry     # no target -> deploy.py (everything)
#        ./deploy.sh waker --limit waker     # -> waker-service/waker.py
#        ./deploy.sh sleeper_api --dry       # -> waker-service/sleeper_api.py
#        ./deploy.sh docker                  # -> deploy_docker.py
#        ./deploy.sh waker-service/waker.py  # full path still works
#        ./deploy.sh --version               # flags pass through untouched
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

if [[ $# -eq 0 || "$1" == -* ]]; then
  resolved="deploy.py"
else
  target="$1"
  shift
  resolved=""
  # An unmatched glob stays literal, and `-f` on a literal '*' is false.
  for candidate in "$target" "deploy_${target}.py" */"${target}.py"; do
    if [[ -f "$candidate" ]]; then
      resolved="$candidate"
      break
    fi
  done
  if [[ -z "$resolved" ]]; then
    echo "deploy.sh: no Deploy file matches '$target' (tried '$target'," \
      "'deploy_${target}.py', '*/${target}.py')" >&2
    echo "available: $(ls deploy_*.py */*.py 2>/dev/null | tr '\n' ' ')" >&2
    exit 1
  fi
fi
set -- "$resolved" "$@"

set -a
# shellcheck source=../.env
source ../.env
# shellcheck source=../waker-service/.env
source ../waker-service/.env
set +a

exec uv run pyinfra inventory.py "$@"
