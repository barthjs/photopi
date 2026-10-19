---
title: Settings
description: Every setting with its environment variable and default, and where the configuration is read from.
sidebar:
  order: 1
---

Every setting can be changed on the configuration page, in the config file, with an environment variable,
or with a command-line flag. [Configuration](../guides/configuration.md) describes the common settings.

## Sources

Each setting is taken from the first source that provides it:

| Priority | Source               | Notes                                                                   |
| -------- | -------------------- | ----------------------------------------------------------------------- |
| 1        | Command-line flag    | Only when actually passed on the command line.                          |
| 2        | Environment variable | Also populated from a `.env` file in the working directory, if present. |
| 3        | Config file          | Optional. See below.                                                    |
| 4        | Built-in default     | Default values.                                                         |

Every setting has one name in three forms:

| Form                 | Rule                                         | Example              |
| -------------------- | -------------------------------------------- | -------------------- |
| Environment variable | Upper case, words joined by `_`              | `LOG_MAX_AGE_DAYS`   |
| Command-line flag    | Lower case, words joined by `-`, `--` prefix | `--log-max-age-days` |
| Config-file key      | Lower case, nested by section                | `log.max_age_days`   |

Every environment variable also has a `_FILE` variant that names a file holding the value, for example
`APP_SECRET_FILE` for secrets of a container. Setting both is an error. All settings are checked at startup,
and every invalid setting is reported.

## Config file

The config file is optional and may be `YAML`, `JSON`, or `TOML`. `--config` or `CONFIG_FILE` sets its path.
Without one, PhotoPi uses the first `config.yaml` it finds in the working directory, in the user configuration
directory under `photopi/`, and in `/etc/photopi/`. The configuration page saves to the file that was loaded,
or creates `config.yaml` in the first writable of these directories. Saved settings take effect after a restart.

## App

| Environment variable | Default      | Description                                                           |
| -------------------- | ------------ | --------------------------------------------------------------------- |
| `APP_NAME`           |              | Application name shown in the interface, empty for the built-in name. |
| `APP_ENV`            | `production` | Environment (development, staging, production).                       |
| `APP_LANGUAGE`       | `en`         | Language of the user interface and the emails (en, de).               |
| `APP_SECRET`         |              | Password of the login, empty allows the login without a password.     |

Secrets such as the password never appear in the logs or in the web interface.

## Log

| Environment variable | Default  | Description                                                                        |
| -------------------- | -------- | ---------------------------------------------------------------------------------- |
| `LOG_LEVEL`          | `warn`   | Log level (debug, info, warn, error).                                              |
| `LOG_CHANNEL`        | `stdout` | Log output channel (stdout, file, both).                                           |
| `LOG_PATH`           |          | Directory of the log files, empty for the log directory of the system or the user. |
| `LOG_FORMAT`         | `json`   | Log format (json, console).                                                        |
| `LOG_MAX_SIZE_MB`    | `100`    | Log file size in megabytes that triggers a rotation before the daily one.          |
| `LOG_MAX_AGE_DAYS`   | `30`     | Days to keep rotated log files, -1 keeps them forever.                             |
| `LOG_COMPRESS`       | `true`   | Compress rotated log files with gzip.                                              |

The log file `photopi.log` rotates every day and when it reaches its maximum size. Without `LOG_PATH`,
it is written to the first writable of these directories:

| OS      | System-wide                  | User                                                                     |
| ------- | ---------------------------- | ------------------------------------------------------------------------ |
| Linux   | `/var/log/photopi`           | `$XDG_STATE_HOME/photopi/logs`, by default `~/.local/state/photopi/logs` |
| Windows | `%ProgramData%\photopi\logs` | `%LOCALAPPDATA%\photopi\logs`                                            |
| macOS   | `/Library/Logs/photopi`      | `~/Library/Logs/photopi`                                                 |

If no directory is writable, PhotoPi logs to the standard output only. A systemd service writes the standard output
to the journal, and the container keeps its log files in `/data/logs`.

## Camera

| Environment variable           | Default   | Description                                                                                                            |
| ------------------------------ | --------- | ---------------------------------------------------------------------------------------------------------------------- |
| `CAMERA_SOURCE`                | `webcam`  | Active camera source (webcam, rpicam, gphoto2).                                                                        |
| `CAMERA_CAPTURE_WIDTH`         | `4000`    | Still capture width in pixels of the rpicam source.                                                                    |
| `CAMERA_CAPTURE_HEIGHT`        | `2400`    | Still capture height in pixels of the rpicam source.                                                                   |
| `CAMERA_PREVIEW_WIDTH`         | `800`     | Live preview width in pixels of the rpicam source.                                                                     |
| `CAMERA_PREVIEW_HEIGHT`        | `480`     | Live preview height in pixels of the rpicam source.                                                                    |
| `CAMERA_PREVIEW_FPS`           | `25`      | Live preview framerate of the rpicam source.                                                                           |
| `CAMERA_OVERLAY_PATH`          |           | Storage path of a PNG overlay composited onto every stored capture, such as overlays/frame.png, empty for none.        |
| `CAMERA_EXPOSURE_TIME`         | `auto`    | Exposure time of the rpicam source with an automatic gain, such as 1/125, auto for a fully automatic exposure.         |
| `CAMERA_EXPOSURE_MODE`         | `normal`  | Profile of the automatic exposure of the rpicam source (normal, short).                                                |
| `CAMERA_ANALOGUE_GAIN`         | `0`       | Fixed analogue gain of the rpicam source, 0 for an automatic gain.                                                     |
| `CAMERA_EXPOSURE_COMPENSATION` | `0`       | Exposure compensation in stops of the rpicam source.                                                                   |
| `CAMERA_WHITE_BALANCE`         | `auto`    | White balance preset of the rpicam source (auto, incandescent, tungsten, fluorescent, indoor, daylight, cloudy).       |
| `CAMERA_HORIZONTAL_FLIP`       | `false`   | Mirror the image of the rpicam source horizontally.                                                                    |
| `CAMERA_VERTICAL_FLIP`         | `false`   | Mirror the image of the rpicam source vertically.                                                                      |
| `CAMERA_ROTATION`              | `0`       | Image rotation in degrees of the rpicam source (0, 180).                                                               |
| `CAMERA_JPEG_QUALITY`          | `93`      | JPEG quality from 1 to 100 of the stills of the rpicam source.                                                         |
| `CAMERA_AUTOFOCUS_MODE`        | `default` | Autofocus mode of a rpicam camera with a focus motor, such as the Camera Module 3 (default, manual, auto, continuous). |
| `CAMERA_DENOISE`               | `auto`    | Noise reduction of the rpicam source (auto, off, cdn_off, cdn_fast, cdn_hq).                                           |

The `rpicam` source requires `rpicam-vid` and `rpicam-still`, and its capture and preview size must have the same
aspect ratio. [Camera](../guides/camera.md) explains the image settings.

## Capture

| Environment variable      | Default   | Description                                                                              |
| ------------------------- | --------- | ---------------------------------------------------------------------------------------- |
| `CAPTURE_IMAGES_PER_RUN`  | `4`       | Number of captures per run.                                                              |
| `CAPTURE_FILE_PREFIX`     | `PhotoPi` | Filename prefix of stored captures.                                                      |
| `CAPTURE_EVENT_NAME`      | `PhotoPi` | Event name in share emails, also the storage folder of the captures.                     |
| `CAPTURE_WELCOME_MESSAGE` |           | Message of the welcome screen, empty for none.                                           |
| `CAPTURE_ADMIN_EMAIL`     |           | Contact address shown to a guest when emailing their captures fails.                     |
| `CAPTURE_SAVE_ORIGINALS`  | `true`    | Store an unmodified copy of each capture under originals/, requires camera.overlay_path. |

## Mail

| Environment variable     | Default | Description                                                                                             |
| ------------------------ | ------- | ------------------------------------------------------------------------------------------------------- |
| `MAIL_DRIVER`            | `log`   | Mail driver, `log` or `smtp`. The `log` driver writes every email to the log instead of sending it.     |
| `MAIL_HOST`              |         | SMTP host of the smtp driver.                                                                           |
| `MAIL_PORT`              | `587`   | SMTP port of the smtp driver.                                                                           |
| `MAIL_USERNAME`          |         | SMTP username of the smtp driver.                                                                       |
| `MAIL_PASSWORD`          |         | SMTP password of the smtp driver.                                                                       |
| `MAIL_FROM_ADDRESS`      |         | Sender email address.                                                                                   |
| `MAIL_FROM_NAME`         |         | Sender display name.                                                                                    |
| `MAIL_REPLY_TO`          |         | Reply-To address of every email, such as a support address next to a no-reply sender, empty for none.   |
| `MAIL_MESSAGE_ID_DOMAIN` |         | Domain of the Message-ID of every email, usually the sending domain, empty lets the SMTP client choose. |

## Print

| Environment variable    | Default   | Description                                                                                                                                             |
| ----------------------- | --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `PRINT_ENABLED`         | `false`   | Print the composed layout of every run through a CUPS queue.                                                                                            |
| `PRINT_PRINTER_NAME`    |           | CUPS queue name as listed by lpstat -p, required when printing is enabled.                                                                              |
| `PRINT_LAYOUT_ID`       | `strip-3` | Layout file without .json that composes the print and share collages.                                                                                   |
| `PRINT_SUBMISSION_MODE` | `managed` | How print jobs reach CUPS, managed lets the printer driver convert and scale the PDF, raw sends it untouched to a printer with its own PDF interpreter. |

[Printing](../guides/printing.md) explains the submission modes and the queue.

## Server

| Environment variable            | Default               | Description                                                                                          |
| ------------------------------- | --------------------- | ---------------------------------------------------------------------------------------------------- |
| `SERVER_HOST`                   | `*`                   | Listen host, * for all interfaces.                                                                   |
| `SERVER_PORT`                   | `8080`                | Listen port.                                                                                         |
| `SERVER_TRUSTED_PROXIES`        | `127.0.0.0/8,::1/128` | Comma-separated CIDRs or IPs of proxies whose forwarding headers are trusted.                        |
| `SERVER_CORS_ALLOWED_ORIGINS`   |                       | Comma-separated origins allowed to call the API from another domain, * for any, empty disables CORS. |
| `SERVER_CORS_ALLOW_CREDENTIALS` | `false`               | Allow cross-origin requests to send cookies, requires explicit origins.                              |
| `SERVER_SSL_ENABLED`            | `false`               | Serve over HTTPS, with a self-signed certificate if ssl_cert_file and ssl_key_file are empty.        |
| `SERVER_SSL_CERT_FILE`          |                       | PEM-encoded TLS certificate file, requires ssl_key_file.                                             |
| `SERVER_SSL_KEY_FILE`           |                       | PEM-encoded private key file of ssl_cert_file.                                                       |

With HTTPS enabled and no certificate files set, PhotoPi generates a self-signed certificate for `localhost`
and its local network addresses and stores it in the `certs` folder of the first writable configuration directory.
Explicit certificate files are used as they are and must exist. [Login and network access](../guides/access.md)
explains HTTPS, the trusted proxies, and the CORS settings.

## Share

| Environment variable             | Default | Description                                                                                                            |
| -------------------------------- | ------- | ---------------------------------------------------------------------------------------------------------------------- |
| `SHARE_EMAIL_ENABLED`            | `true`  | Allow guests to email themselves their captures.                                                                       |
| `SHARE_EMAIL_SUBJECT`            |         | Subject of the share email, empty for the built-in subject.                                                            |
| `SHARE_CLOUD_DRIVER`             |         | Cloud share driver (nextcloud, self), empty disables the cloud share.                                                  |
| `SHARE_CLOUD_NEXTCLOUD_URL`      |         | Nextcloud base URL of the nextcloud driver.                                                                            |
| `SHARE_CLOUD_NEXTCLOUD_USERNAME` |         | Nextcloud username of the nextcloud driver.                                                                            |
| `SHARE_CLOUD_NEXTCLOUD_PASSWORD` |         | Nextcloud app password of the nextcloud driver.                                                                        |
| `SHARE_CLOUD_NEXTCLOUD_FOLDER`   |         | Nextcloud folder the nextcloud driver uploads into.                                                                    |
| `SHARE_CLOUD_SELF_BASE_URL`      |         | Public base URL guests reach this server at, for the QR code link of the self driver, such as http://192.168.4.1:8080. |

The `nextcloud` driver requires the URL, the username, the password, and the folder. The `self` driver requires
the base URL. [Sharing](../guides/sharing.md) explains both.

## Storage

| Environment variable        | Default | Description                                                               |
| --------------------------- | ------- | ------------------------------------------------------------------------- |
| `STORAGE_DRIVER`            | `local` | Storage driver (local, s3).                                               |
| `STORAGE_PATH`              |         | Storage root directory of the local driver, empty for the data directory. |
| `STORAGE_S3_ENDPOINT`       |         | S3 endpoint URL of the s3 driver.                                         |
| `STORAGE_S3_REGION`         |         | S3 region of the s3 driver.                                               |
| `STORAGE_S3_BUCKET`         |         | S3 bucket of the s3 driver.                                               |
| `STORAGE_S3_ACCESS_KEY`     |         | S3 access key of the s3 driver.                                           |
| `STORAGE_S3_SECRET_KEY`     |         | S3 secret key of the s3 driver.                                           |
| `STORAGE_S3_USE_PATH_STYLE` | `true`  | Use path-style bucket addressing.                                         |

The `s3` driver requires the endpoint, the bucket, the access key, and the secret key. Without `STORAGE_PATH`,
the `local` driver stores its files in the first writable of these directories:

| OS      | System-wide                            | User                                                          |
| ------- | -------------------------------------- | ------------------------------------------------------------- |
| Linux   | `/var/lib/photopi`                     | `$XDG_DATA_HOME/photopi`, by default `~/.local/share/photopi` |
| Windows | `%ProgramData%\photopi`                | `%LOCALAPPDATA%\photopi`                                      |
| macOS   | `/Library/Application Support/photopi` | `~/Library/Application Support/photopi`                       |

The storage contains the photos in `captures/`, the originals in `originals/`, the collages in `prints/`,
the overlays in `overlays/`, and the layouts in `layouts/`, each with a folder per event where it applies.

## System

| Environment variable            | Default                                                          | Description                                                                                               |
| ------------------------------- | ---------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| `SYSTEM_SERVER_RESTART_COMMAND` | `systemctl --user restart photopi.service`                       | Shell command that restarts the service of the application, empty disables the button.                    |
| `SYSTEM_SERVER_STOP_COMMAND`    | `systemctl --user stop photopi.service`                          | Shell command that stops the service of the application, empty disables the button.                       |
| `SYSTEM_KIOSK_RESTART_COMMAND`  | `systemctl --user restart photopi-kiosk.service`                 | Shell command that restarts the service of the kiosk browser, empty disables the button.                  |
| `SYSTEM_KIOSK_STOP_COMMAND`     | `systemctl --user stop photopi-kiosk.service`                    | Shell command that stops the service of the kiosk browser, empty disables the button.                     |
| `SYSTEM_ALL_RESTART_COMMAND`    | `systemctl --user restart photopi.service photopi-kiosk.service` | Shell command that restarts the application and the kiosk browser in one step, empty disables the button. |
| `SYSTEM_ALL_STOP_COMMAND`       | `systemctl --user stop photopi-kiosk.service photopi.service`    | Shell command that stops the application and the kiosk browser in one step, empty disables the button.    |

[System](../guides/system.md#system-page) explains the commands.

## Theme

| Environment variable | Default | Description                                          |
| -------------------- | ------- | ---------------------------------------------------- |
| `THEME_PRIMARY`      |         | Primary color (hex), empty for the built-in color.   |
| `THEME_SECONDARY`    |         | Secondary color (hex), empty for the built-in color. |
| `THEME_SUCCESS`      |         | Success color (hex), empty for the built-in color.   |
| `THEME_INFO`         |         | Info color (hex), empty for the built-in color.      |
| `THEME_WARNING`      |         | Warning color (hex), empty for the built-in color.   |
| `THEME_ERROR`        |         | Error color (hex), empty for the built-in color.     |
| `THEME_NEUTRAL`      |         | Neutral color (hex), empty for the built-in color.   |

The theme colors the web interface.
