#!/bin/sh
# Sets blackberry up for hostd's stack sync; run as root by `task stacks`.
# Idempotent: keys are only generated when missing, the clone only made once.
#   stacks-setup.sh keys   install git, age and sops; generate the age key and
#                          the deploy key; print both public keys
#   stacks-setup.sh clone  clone deploys to /srv/deploys (the deploy key must
#                          be registered on GitHub first)
set -eu

SOPS_VERSION=v3.13.3
SOPS_SHA256=e5bec3346a873ae91d871550f3e698c1aad962aff462a080e40f25fde17fef6b
REPO=git@github.com:nikbogman/deploys.git
DIR=/srv/deploys
SSH="ssh -i /etc/hostd/deploy_key -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new -o UserKnownHostsFile=/etc/hostd/known_hosts"

case ${1:-} in
keys)
  command -v git >/dev/null && command -v age-keygen >/dev/null ||
    DEBIAN_FRONTEND=noninteractive NEEDRESTART_MODE=l apt-get install -y -qq git age >/dev/null 2>&1
  if [ "$(sops --version --disable-version-check 2>/dev/null | awk 'NR==1{print "v"$2}')" != "$SOPS_VERSION" ]; then
    curl -fsSLo /tmp/sops "https://github.com/getsops/sops/releases/download/$SOPS_VERSION/sops-$SOPS_VERSION.linux.amd64"
    echo "$SOPS_SHA256  /tmp/sops" | sha256sum -c --quiet
    install -m755 /tmp/sops /usr/local/bin/sops && rm -f /tmp/sops
  fi
  install -d -m700 /etc/hostd
  [ -f /etc/hostd/age.key ] || age-keygen -o /etc/hostd/age.key 2>/dev/null
  chmod 600 /etc/hostd/age.key
  [ -f /etc/hostd/deploy_key ] || ssh-keygen -q -t ed25519 -N '' -C hostd@blackberry -f /etc/hostd/deploy_key
  echo "age $(age-keygen -y /etc/hostd/age.key)"
  echo "deploy $(cat /etc/hostd/deploy_key.pub)"
  ;;
clone)
  if [ ! -d "$DIR/.git" ]; then
    GIT_SSH_COMMAND="$SSH" git clone -q "$REPO" "$DIR"
  fi
  git -C "$DIR" config core.sshCommand "$SSH"
  chmod 700 "$DIR" # decrypted .env files live here
  ;;
*)
  echo "usage: $0 keys|clone" >&2
  exit 1
  ;;
esac
