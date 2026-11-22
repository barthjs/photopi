---
title: Windows
description: Install PhotoPi on Windows with the installation script or from a release archive.
sidebar:
  order: 3
---

On Windows, PhotoPi runs as a desktop application or as a Windows service.

## Installation script

The installation script requires PowerShell 7. It downloads the latest release, verifies it against the published
checksums, installs it, and creates a shortcut in the start menu:

```powershell
Invoke-WebRequest https://raw.githubusercontent.com/barthjs/photopi/main/scripts/install.ps1 -OutFile install.ps1
pwsh -File install.ps1 -Mode User
```

| Mode     | Binary                                   | Working directory       |
| -------- | ---------------------------------------- | ----------------------- |
| `User`   | `%LOCALAPPDATA%\Programs\photopi`        | `%APPDATA%\photopi`     |
| `System` | `%ProgramFiles%\photopi`, requires admin | `%ProgramData%\photopi` |

Without `-Mode`, the script asks for the mode.

## Windows service

The script offers to install a Windows service named `PhotoPi`, which starts with Windows and restarts PhotoPi
if it stops. The service requires [NSSM](https://nssm.cc), which can be installed with `winget install nssm`.
Without NSSM, the script skips the service, and running it again after installing NSSM adds the service.

## Release archive

Every release has a ZIP archive per architecture, which holds `photopi.exe`:

| Architecture | Archive                                                                                                            |
| ------------ | ------------------------------------------------------------------------------------------------------------------ |
| x86_64       | [photopi_windows_amd64.zip](https://github.com/barthjs/photopi/releases/latest/download/photopi_windows_amd64.zip) |
| ARM64        | [photopi_windows_arm64.zip](https://github.com/barthjs/photopi/releases/latest/download/photopi_windows_arm64.zip) |
| x86 (32-bit) | [photopi_windows_386.zip](https://github.com/barthjs/photopi/releases/latest/download/photopi_windows_386.zip)     |

Unpack it into a directory of your choice and start `photopi.exe`.
A double click on its icon in the notification area opens the web interface in the browser, and a right click opens the menu.

## Files

PhotoPi looks for its config file in the working directory first. It stores its data in `%ProgramData%\photopi`
if that directory is writable, otherwise in `%LOCALAPPDATA%\photopi`.
[Settings](../reference/settings.md#config-file) describes the order in detail.

## Updates

PhotoPi updates itself from the web interface, see [Updates](./updates.md). If it may not write to its own file,
the web interface says so, and running the installation script again installs the new version.
