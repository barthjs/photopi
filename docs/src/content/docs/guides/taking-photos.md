---
title: Taking photos
description: How guests take photos and how PhotoPi stores them.
sidebar:
  order: 2
---

Guests use PhotoPi on the kiosk screen or any device with a browser. A run starts on the welcome screen and takes
a number of photos with a countdown. The review screen shows them, and the share screen offers to send them
by email, to open them on a phone, or to print them.

The Capture tab of the configuration page sets the number of photos per run, the welcome message, and the name
of the event, which appears in the emails and the layouts and names the folder of the photos.

## Runs

- The booth serves one guest at a time. A run belongs to the browser that started it, and only that browser can take
  its photos, see them, and share them.
- While a run takes photos, no other device can start one, until the run is complete or has been idle for two minutes.
- A run left incomplete, for example after a reload of the page, is deleted when the next run starts.
- When the countdown ends, the kiosk screen turns white as the light of the photo in a dark booth, and then shows
  the photo for a moment while the live preview connects again.

## Storage of the photos

- The photos of an event are stored in one folder, named after the event.
- The file names start with the run number, so the folder sorts by run even if the clock of a booth without internet
  is wrong.
- Run numbers never repeat, also after a run is deleted.
- Renaming the event only affects photos taken afterward.

## Originals

With an overlay active and the originals enabled, PhotoPi also stores every photo without the overlay. Guests never
receive the originals. The gallery can show and download them after the login.
