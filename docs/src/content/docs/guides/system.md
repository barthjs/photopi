---
title: System
description: Watch the computer of the booth and restart or shut it down from the web interface.
sidebar:
  order: 8
---

The dashboard and the System page show the state of the computer of the booth and control it from the web interface.

## Dashboard

The dashboard shows the host name, the network addresses, and the use of processor, memory, and disk
of the computer, and how long PhotoPi has been running. The processor use is measured once when the page loads.
The disk use covers the root file system, so photos on S3 or another drive are not part of it.

## System page

The System page offers these actions:

| Action                            | Command                                              |
| --------------------------------- | ---------------------------------------------------- |
| Restart or stop PhotoPi           | Set in the System section of the configuration page. |
| Restart or stop the kiosk browser | Set in the System section of the configuration page. |
| Restart or stop both              | Set in the System section of the configuration page. |
| Exit PhotoPi                      | Ends the process. A service starts it again.         |
| Reboot or shut down the computer  | The command of the operating system.                 |

The defaults of the commands match the user services of the installation script:

```sh
systemctl --user restart photopi.service photopi-kiosk.service
systemctl --user stop photopi-kiosk.service photopi.service
```

- The services of the Debian package and of the system-wide installation cannot restart themselves, so they disable
  these commands. An empty command disables its button.
- A command runs with `sh -c`, so it can also be a script. A command that fails is logged with its output.
- While an update is installed, the buttons that stop or restart are rejected.
