#!/bin/sh
set -e

if ! getent passwd photopi >/dev/null; then
  useradd --system --user-group --home-dir /var/lib/photopi --no-create-home --shell /usr/sbin/nologin photopi
fi

mkdir -p /etc/photopi /var/lib/photopi
chown -R photopi:photopi /etc/photopi /var/lib/photopi

systemctl daemon-reload >/dev/null 2>&1 || true

if systemctl is-enabled --quiet photopi 2>/dev/null; then
  systemctl restart photopi >/dev/null 2>&1 || true
fi

cat <<'MSG'
PhotoPi installed.

Configure it at /etc/photopi/config.yaml,
then enable and start it with:

  sudo systemctl enable --now photopi

A Chromium kiosk browser unit is also included, as a per-user systemd service.
As the user who will run the kiosk, in their own graphical session:

  systemctl --user enable --now photopi-kiosk
MSG

exit 0
