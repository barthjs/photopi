---
title: Gallery
description: Browse, download, and delete the photos of past runs.
sidebar:
  order: 6
---

The gallery shows the runs of one event as a timeline, the most recent first,
grouped by the day, week, month, or year they were taken in, and switches between the events:

| Action             | Result                                                                                  |
| ------------------ | --------------------------------------------------------------------------------------- |
| Open a run         | Shows its photos and the [originals](./taking-photos.md#originals) without the overlay. |
| Download a photo   | The shown photo, with or without the overlay as the view of the run shows it.           |
| Download a run     | A zip file with the photos of the run, named after the event and the run.               |
| Download selection | A zip file with the photos of every selected run, at most 200 runs.                     |
| Download an event  | A zip file with the photos of every run of the event.                                   |
| Delete a run       | Removes its photos and originals from the storage. Files on Nextcloud stay there.       |
| Delete selection   | Removes every selected run like a single one and reports the runs that failed.          |

The tile of a run shows a small copy of its first photo and the number of its photos if it has several.
In the view of a run, the button with the "i" opens its event, the time it was taken, and the number of its photos and originals.

To choose runs, hover a tile and tick its checkbox, or press it for a moment on a touch screen.
While runs are chosen, a click chooses a run instead of opening it, a click with the shift key chooses every run
up to the last chosen one, and Escape clears the choice. The bar at the bottom holds the downloads and the deletion.

The time of a run comes from the file names of its photos, and the gallery shows it as the booth recorded it
in every time zone. The booth takes this time from its own clock, which can be wrong on a device without internet.
The gallery orders the runs by this time, so a run follows its day even after the clock went back,
and runs without a readable time come last.
