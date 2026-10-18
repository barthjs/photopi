#!/bin/sh
set -e

if [ "$1" = "remove" ] || [ "$1" = "purge" ]; then
  systemctl disable photopi >/dev/null 2>&1 || true
fi

systemctl daemon-reload >/dev/null 2>&1 || true

exit 0
