---
title: Configuration
description: Change the settings of PhotoPi.
sidebar:
  order: 9
---

Most installations need only a few settings: a password for the login, the camera, and the way guests receive
their photos. Everything else has a default that suits a typical event.

## Changing settings

The configuration page of the web interface changes every setting. It saves them to the config file,
and they take effect after PhotoPi restarts. Settings can also be set in the config file by hand,
or with environment variables and command-line flags, which take precedence over the file. A field set
by an environment variable or a flag shows a warning, since saving it in the file has no effect.

[Settings](../reference/settings.md) lists every setting with its environment variable and default.

## Common settings

| Setting        | Environment variable     | Purpose                                                               |
| -------------- | ------------------------ | --------------------------------------------------------------------- |
| Password       | `APP_SECRET`             | Protects the pages that need a login, see [Login](./access.md#login). |
| Language       | `APP_LANGUAGE`           | Language of the web interface.                                        |
| Camera source  | `CAMERA_SOURCE`          | The camera that takes the photos, see [Camera](./camera.md).          |
| Photos per run | `CAPTURE_IMAGES_PER_RUN` | Number of photos a guest takes in one run.                            |
| Event name     | `CAPTURE_EVENT_NAME`     | Name of the event in the emails and the layouts.                      |
| Port           | `SERVER_PORT`            | Port of the web interface, 8080 by default.                           |

Passwords and other secrets are never shown again after saving. Leaving their field empty keeps them.

## Colors

The Theme tab of the configuration page changes the colors of the web interface.
