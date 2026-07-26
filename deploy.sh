#!/usr/bin/env bash
# Forwards to provisioning/deploy.sh so a Deploy can be run from the repo
# root without `cd provisioning` first. All args pass through untouched;
# see provisioning/deploy.sh for the actual logic and usage, and the repo
# root .env for the secrets/config it loads.
set -euo pipefail
exec "$(dirname "${BASH_SOURCE[0]}")/provisioning/deploy.sh" "$@"
