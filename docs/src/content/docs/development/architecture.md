---
title: Architecture
description: How the backend, the web interface, and the documentation are structured.
sidebar:
  order: 2
---

PhotoPi is one Go binary. It controls the camera, stores the photos, sends them to guests, and serves the web interface
and this documentation, which are built beforehand and embedded.

## Layers

Requests and commands pass through three layers, and a layer only calls the layer below it:

| Layer    | Package                        | Role                                                                |
| -------- | ------------------------------ | ------------------------------------------------------------------- |
| Input    | `internal/api`, `internal/cli` | Read and validate requests and commands, then call a service.       |
| Services | `internal/service`             | Hold the rules and call the stores and the infrastructure packages. |
| Stores   | `internal/store`               | Read and write the data.                                            |

The services cover auth, capture runs, commands, configuration, the gallery, layouts, overlays, printing,
sharing, the system, and updates.

Domain types in `internal/domain` are only used inside the backend. The API reads request types and writes response
types, so its format does not change when the internal model does.

## Assembly

`internal/app` is the only place that wires the layers and the packages together. It offers building blocks, such as
the file storage and the stores, and `app.New` assembles the whole server from them. The server and the feature tests
start the application through `app.New`. A command that needs only some parts builds just those, so it does not open
the camera or start the HTTP API.

The configuration in `internal/config` reads the settings from their sources. The services and the API read
no configuration themselves. They take their own options, which `internal/app` fills from the configuration.
Only the service and the handlers that manage the configuration itself use its types.

## Packages

Self-contained parts of the domain have packages of their own, since they import nothing from the API or the services:

| Package           | Role                                                                 |
| ----------------- | -------------------------------------------------------------------- |
| `internal/camera` | Streams the live preview and takes the photos of the camera sources. |
| `internal/share`  | Uploads the photos of a run to a cloud provider.                     |

Infrastructure packages hold no business logic and import neither `config` nor the domain types:

| Package                | Role                                                             |
| ---------------------- | ---------------------------------------------------------------- |
| `internal/server`      | Runs the HTTP server with its middleware and security headers.   |
| `internal/media`       | Stores files locally or in S3.                                   |
| `internal/session`     | Keeps the sessions of the login in memory.                       |
| `internal/ratelimiter` | Limits requests in fixed windows.                                |
| `internal/health`      | Answers the liveness and readiness probes.                       |
| `internal/logging`     | Writes the log to the standard output, a rotating file, or both. |
| `internal/update`      | Replaces the binary with the latest release.                     |
| `internal/i18n`        | Translates the texts the backend produces.                       |
| `internal/crypto`      | Wraps cryptographic primitives behind small functions.           |
| `internal/mailer`      | Renders and sends localized emails.                              |

## Conventions

Constructors and functions take their parameters in this order:

1. `ctx context.Context`
2. the logger
3. configuration values or an options struct of the package
4. infrastructure dependencies, such as the storage
5. services and stores
6. callbacks, such as a function that stops the application
7. functional options as the variadic last parameter

Services use the concrete stores. An interface only exists where there are several implementations, such as
the storage drivers, or where a test replaces something outside the process, such as the camera or the mail transport.

## Web interface

The web interface in `web/` is a Vue application with Nuxt UI and Tailwind CSS:

| Folder            | Content                                                                       |
| ----------------- | ----------------------------------------------------------------------------- |
| `src/api`         | The client of the API with one repository per resource.                       |
| `src/pages`       | One component per page, with the layouts in `src/layouts`.                    |
| `src/components`  | Components by feature, with reusable blocks such as `table` and `form`.       |
| `src/composables` | Reusable logic of the components.                                             |
| `src/lib`         | Functions without Vue, such as the targets of the documentation in `docs.ts`. |
| `src/stores`      | Pinia stores for state that several pages share.                              |
| `src/locales`     | The texts of the interface in every language.                                 |

The screens for the guests run in the kiosk browser or on any device, and the admin pages manage the booth.
Everything goes through the JSON API under `/api`, which the [API reference](/api/) describes,
except what the server announces itself through the websocket under `/api/ws`:

| Message  | Sent                                            | Data                                                           |
| -------- | ----------------------------------------------- | -------------------------------------------------------------- |
| `hello`  | on connect                                      | the version and a hash of the settings the web interface reads |
| `health` | on connect and when the camera fails or returns | the state of every dependency, as `/readyz` names it           |

- A page reloads once `hello` names another version or other settings, as soon as a reload loses nothing
  and nobody reads the page, since it is hidden or its route sets `meta.unattended`. Until then it offers the reload,
  and the next navigation loads a new page. Unsaved changes that `useUnloadGuard` or `useDiscardConfirm` watch,
  and every other `useReloadHold`, keep it from reloading on its own.
- An uncaught error shows a screen that reloads the page after 10 seconds, and after twice as long for every error
  that returns. A navigation that finds a module of an older build missing loads its target as a new page.
- A navigation without an answer from the server, or with a gateway error of a reverse proxy in front of it,
  shows the offline page of the service worker, which tries again every 10 seconds.

## Translations

The web interface and the backend are translated into every supported language:

| Texts                           | Files                                         | Test                             |
| ------------------------------- | --------------------------------------------- | -------------------------------- |
| Web interface                   | `web/src/locales/*.json`                      | `web/src/locales.test.ts`        |
| Backend, such as error messages | `internal/i18n/locales/*.json`                | `TestLocales` in `internal/i18n` |
| Documentation                   | see [Languages](./documentation.md#languages) |                                  |

- English is the default language.
- Every text exists in every language with the same keys, and the tests fail for a missing key.
- A new or changed text updates every language in the same change.
- German texts address the user with "du". Texts under shared keys, such as `errors.*` and `common.*`, stay generic.

## Documentation

The documentation in `docs/` is an Astro site with Starlight, built into `docs/dist` and embedded into the binary
like the web interface. [Documentation](./documentation.md) describes how it is written and built.
