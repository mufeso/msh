#!/bin/bash
# health-check.sh - a list of read-only checks to run on many servers:
#
#   msh script run --group web --file examples/scripts/health-check.sh
#   msh script run --all --file examples/scripts/health-check.sh --arg nginx
#
# msh streams this file to "bash -s" over SSH; nothing is copied to the
# servers. Arguments arrive as "$1", "$2", ...
set -u

service="${1:-sshd}"

echo "== host"
hostname
uptime

echo "== disk (over 80%)"
df -hP | awk 'NR == 1 || $5 + 0 > 80'

echo "== memory"
free -m 2>/dev/null || echo "free not available"

echo "== failed units"
systemctl --failed --no-pager --no-legend 2>/dev/null || echo "systemd not available"

echo "== service ${service}"
systemctl is-active "$service" 2>/dev/null || echo "inactive or unknown"
