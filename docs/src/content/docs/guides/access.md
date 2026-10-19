---
title: Login and network access
description: Protect the web interface, serve it over HTTPS, and run it behind a reverse proxy.
sidebar:
  order: 10
---

PhotoPi is meant for a local network. This page describes the login, HTTPS, and what to set when a reverse proxy
or another website sits in front of it.

## Login

| Pages                                                             | Login        |
| ----------------------------------------------------------------- | ------------ |
| Dashboard, configuration page, gallery, Layouts page, System page | Required     |
| Screens for the guests, shared links                              | Not required |

- The password is the setting `APP_SECRET`. Without it, the login accepts an empty password.
- Set a password whenever PhotoPi is reachable from a network you do not control.
- The password also signs the shared links, so they stay valid across a restart only with a password.

## HTTPS

- With HTTPS enabled and no certificate files set, PhotoPi generates a self-signed certificate for its addresses
  in the local network, which browsers warn about.
- The `webcam` source needs HTTPS when the capture screen runs on another device than the booth, since browsers
  only allow the camera on `localhost` or a secure page.
- [Server](../reference/settings.md#server) describes the settings.

## Reverse proxy

- PhotoPi takes the address of the client from the forwarding headers only if the proxy is trusted. By default,
  only proxies on the same computer are trusted.
- A proxy on another host or in a container network must be listed in `SERVER_TRUSTED_PROXIES`. Otherwise all clients
  share its address and its rate limits.
- The proxy should pass the security headers of PhotoPi through.
- A proxy that serves only HTTPS may add `Strict-Transport-Security`.

## Websites that use the API

- A website on another domain can only use the API in the browser if its origin is listed in
  `SERVER_CORS_ALLOWED_ORIGINS`.
- `*` lets every origin read data, while requests that change data need the origin listed explicitly.
- Programs outside the browser, such as curl, are not affected.
