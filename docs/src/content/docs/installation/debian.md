---
title: Debian package
description: Install PhotoPi as a system service on Debian and Ubuntu.
sidebar:
  order: 4
---

The Debian package installs PhotoPi with a system service. Download the package for the architecture
of the computer from the [releases page](https://github.com/barthjs/photopi/releases/latest) and install it:

```sh
sudo apt install ./photopi_*.deb
```

The service runs as the system user `photopi`.

| Path                       | Content                |
| -------------------------- | ---------------------- |
| `/etc/photopi/config.yaml` | The config file.       |
| `/var/lib/photopi`         | The data of PhotoPi.   |
| `/var/log/photopi`         | Log files, if enabled. |

## Configure and start

Create a config file with every setting at its default, adjust it, then enable and start the service:

```sh
sudo -u photopi photopi config init /etc/photopi/config.yaml
sudo editor /etc/photopi/config.yaml
sudo systemctl enable --now photopi
```

The settings can also be changed in the web interface. They take effect after a restart
with `sudo systemctl restart photopi`. The service writes its log to the journal:

```sh
journalctl -u photopi
```

The package also contains the kiosk browser, see [Kiosk](../guides/kiosk.md).

## Updates

Download the package of the new release and install it the same way. The web interface reports a newer release
and links to it, but does not install it.
