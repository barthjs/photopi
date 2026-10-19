---
title: Sharing
description: Let guests send their photos by email or open them on a phone.
sidebar:
  order: 3
---

The share screen offers guests to send their photos by email and shows a QR code that opens them on a phone.
Both are set in the Share tab of the configuration page. Emails and uploads run in the background, and the share
screen shows their result. If the mail server or the cloud cannot be reached, the share fails right away,
since a booth often runs without internet. It is not sent later.

## Email

The email share sends the photos of a run to an address the guest enters. It uses the settings of the Mail tab:
an SMTP server with its sender address, or the `log` driver, which only writes the emails to the log.
If sending fails, the guest sees the contact address of the Capture tab with the run number.

## Cloud share

The cloud share shows guests a QR code with a link to their photos:

| Driver      | Link                                                                           |
| ----------- | ------------------------------------------------------------------------------ |
| Nextcloud   | PhotoPi uploads the photos into a folder of a Nextcloud account and shares it. |
| Self-hosted | PhotoPi serves the photos itself for seven days under a signed link.           |

The self-hosted link uses the public base URL, which has to be an address the phones of the guests can reach:
the address of the booth in the Wi-Fi of an event without internet, or a public domain behind a reverse proxy.
The link is signed with the password of the login, so it stays valid across a restart only if a password is set.

The Test connection button checks the entered values before they are saved.

## Collages

While [printing](./printing.md) is enabled, the emails and the cloud share also contain the collage of the run
in the selected layout.
