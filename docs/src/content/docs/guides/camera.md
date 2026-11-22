---
title: Camera
description: Choose the camera source and adjust the image to the venue.
sidebar:
  order: 1
---

PhotoPi takes photos with one of three camera sources, set in the Camera section of the configuration page:

| Source    | Camera                                                                                       |
| --------- | -------------------------------------------------------------------------------------------- |
| `webcam`  | The camera of the device whose browser shows the capture screen. This is the default.        |
| `rpicam`  | The camera module of a Raspberry Pi, through `rpicam-vid` and `rpicam-still`.                |
| `gphoto2` | A camera connected by USB, through `gphoto2`. Image size and exposure are set on the camera. |

If the camera fails, the welcome screen disables the start button and shows a notice, while the gallery
and shared links keep working.

## Installing the camera programs

The `webcam` source needs nothing on the computer of PhotoPi. The browser asks for access to the camera, which it only
allows on `localhost` or over [HTTPS](./access.md#https). It takes photos in the largest size the camera offers, up to 4K.

The Debian package installs `gphoto2` and recommends `rpicam-apps`, so `apt` installs both with it. Other installations
install them by hand.

The `rpicam` source needs `rpicam-vid` and `rpicam-still` on a Raspberry Pi with its camera module connected.
Raspberry Pi OS usually includes them. If they are missing, install them with:

```sh
sudo apt install rpicam-apps
```

The `gphoto2` source needs `gphoto2` and a camera that gphoto2 supports, connected by USB:

```sh
sudo apt install gphoto2
```

The user that runs PhotoPi needs access to the camera through the groups `video` and `plugdev`. The service
of the Debian package gets both, and the user of a Raspberry Pi OS desktop is already a member. A desktop that opens
a USB camera as a drive blocks it for `gphoto2`, so the camera has to be ejected there first.

## Image settings

The image settings of the `rpicam` source apply to the live preview and to the photos, so the preview shows what
the photo will look like. They depend on the light of the venue and on how the camera is mounted:

| Setting               | Use                                                                                                                               |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| Exposure time         | Automatic, or fixed such as 1/125 s. A short time keeps moving guests sharp but darkens the photo, 1/60 s is a common compromise. |
| Exposure mode         | With an automatic exposure time, Short keeps the time short and raises the gain earlier.                                          |
| Analogue gain         | Automatic, or fixed for a venue with constant light.                                                                              |
| Exposure compensation | Brightens or darkens an automatic exposure in stops.                                                                              |
| Denoise               | Reduces the noise of a high gain.                                                                                                 |
| White balance         | Adjusts the colors to the light of the venue.                                                                                     |
| Flip and rotation     | Mirror or turn the image for a camera behind a mirror or mounted upside down.                                                     |
| Autofocus             | Only for a camera with a focus motor.                                                                                             |

- More light at the booth makes the photos both sharper and brighter.
- The capture and the preview size must have the same aspect ratio.
- On a Raspberry Pi 5, a lower preview framerate leaves more processing power for the photos.

## Overlay

An overlay is a PNG file drawn over every photo, for example a frame with the name of the event. The overlay library
in the Camera section uploads and manages the files, and the selected overlay is scaled to the size of each photo.
With an overlay active, PhotoPi can also keep the photos without it, see [Taking photos](./taking-photos.md#originals).

## Container

The container image contains `gphoto2`, so a camera connected by USB works once the USB devices of the host
are passed into the container:

```yaml
services:
  photopi:
    privileged: true
    volumes:
      - /dev/bus/usb:/dev/bus/usb
```

The `rpicam` source does not work in the container, since its programs are not available for Alpine, the base
of the image. The `webcam` source needs nothing in the container, since it uses the camera of the browser.
