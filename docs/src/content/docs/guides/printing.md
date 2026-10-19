---
title: Printing
description: Print the photos of a run through a CUPS printer.
sidebar:
  order: 4
---

With printing enabled, the share screen shows a Print button that prints the photos of the run in the selected
[layout](./layouts.md). The Printing tab of the configuration page selects the queue, the layout, and the mode.

## Setting up the printer

PhotoPi prints through CUPS, the print system of Linux, to a queue that is set up on the computer with the driver
of the printer. Install CUPS, for example on Debian with:

```sh
sudo apt install cups
```

Then add the printer in the printer settings of the desktop or in the web interface of CUPS at
`http://localhost:631`. Photo printers often need an extra driver package, such as `printer-driver-gutenprint`.
`lpstat -p` lists the queues with the names that PhotoPi shows for selection.

## Submission mode

PhotoPi renders the pages of the layout into one PDF with the paper size of the layout and hands it to the queue:

| Mode      | Use                                                                                                                                   |
| --------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| `managed` | The default. CUPS and the driver scale the print to the paper, so the paper size of the layout has to match a size the driver offers. |
| `raw`     | The PDF is sent untouched, so only printers that process PDF files themselves can print it.                                           |

`raw` suits office printers with PDF support whose driver gets the size wrong. A dedicated photo printer
with its own paper size uses `managed`.

## Queue

Before a print starts, PhotoPi checks the queue. A queue that is missing, disabled, or already holds several jobs
refuses the print, and the guest sees that the printer is not ready. Test connection checks the queue without printing.

CUPS stops a queue when a job fails, for example after a paper jam. To let it drop the failed job and continue
with the next one, set its error policy once:

```sh
lpadmin -p <queue> -o printer-error-policy=abort-job
```

The Printing tab can resume a stopped queue and cancel its jobs. Both need a user that may manage CUPS queues,
such as a member of the `lpadmin` group. For the service of the Debian package, followed by a restart:

```sh
sudo usermod -aG lpadmin photopi
```
