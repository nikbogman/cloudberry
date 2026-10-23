#!/bin/sh
# Run as root on blackberry by `task network`. Installs /tmp/netplan.yaml and
# applies it with a safety net: unless `task network` cancels the
# netplan-rollback timer within 2 minutes (it does once a fresh ssh session
# gets in), the previous config is restored and re-applied.
set -e
cfg=/etc/netplan/00-installer-config.yaml
bak=/root/netplan-00-installer-config.yaml.bak
cp "$cfg" "$bak"
install -m600 /tmp/netplan.yaml "$cfg"
rm -f /tmp/netplan.yaml /tmp/netplan-apply.sh
netplan generate
systemctl stop netplan-rollback.timer 2>/dev/null || true
systemd-run --quiet --unit=netplan-rollback --on-active=120 \
  sh -c "install -m600 $bak $cfg && netplan apply"
# Detached, so this ssh session can return before the network reconfigures.
nohup sh -c 'sleep 1; netplan apply' >/dev/null 2>&1 &
