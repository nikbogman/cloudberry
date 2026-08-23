#!/usr/bin/env bash
# Forwards to deploy/deploy.sh so a Deploy can be run from the repo
# root without `cd deploy` first. All args pass through untouched;
# see deploy/deploy.sh for the actual logic and usage, and the repo
# root .env for the secrets/config it loads.
set -euo pipefail
exec "$(dirname "${BASH_SOURCE[0]}")/deploy/deploy.sh" "$@"
