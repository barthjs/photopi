---
title: Kiosk
description: Run the booth in full screen and start it with the computer.
sidebar:
  order: 7
---

At an event, PhotoPi runs on the computer of the booth, and its screen shows the screens for the guests: the welcome
screen, the countdown, the review, and the share screen. The admin pages are opened from another device
in the network, such as a phone or a laptop. The screens for the guests are made for touch screens down to
800 x 480 pixels.

The kiosk browser shows PhotoPi in Chromium in full screen, without an address bar or other windows, so guests
only see the booth. It opens `http://localhost:8080` and waits until PhotoPi answers, and it starts again
if it is closed.

## How it runs

The kiosk browser needs the screen, which only the user logged in on the desktop has access to. It therefore runs
as the user service `photopi-kiosk.service` of that user, while PhotoPi itself can run as a system service or as
a user service as well. A user service starts with the login of its user, so with an automatic login of the desktop,
the booth starts on its own when the computer is switched on. On Raspberry Pi OS, `raspi-config` enables
the automatic login under System Options.

The kiosk browser needs Chromium and `curl`. The Debian package recommends both, so `apt` installs them with it.
Otherwise install them, on Debian for example with:

```sh
sudo apt install chromium curl
```

## Setting it up

| Installation                                               | Kiosk browser                                                      |
| ---------------------------------------------------------- | ------------------------------------------------------------------ |
| [Debian package](../installation/debian.md)                | Included, enabled by the user logged in on the desktop.            |
| [Installation script](../installation/linux.md), user-wide | Offered during the installation.                                   |
| Installation script, system-wide                           | Not available, since a system installation has no desktop session. |
| Windows and container                                      | Not available. Open PhotoPi in a browser in full screen instead.   |

With the Debian package, the user logged in on the desktop enables the kiosk browser once:

```sh
systemctl --user enable --now photopi-kiosk
```

The user-wide installation script installs PhotoPi and the kiosk browser as user services and adds three launchers
to the application menu: PhotoPi starts both, PhotoPi Server only PhotoPi, and PhotoPi Kiosk only the kiosk browser,
which starts PhotoPi with it.

## Leaving the kiosk

The System page restarts and stops the kiosk browser, see [System](./system.md#system-page). On the computer itself,
the kiosk browser stops with:

```sh
systemctl --user stop photopi-kiosk
```
