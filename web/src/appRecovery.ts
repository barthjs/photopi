import type { App } from 'vue';
import { useEventListener } from '@vueuse/core';

import { metaRepository } from '@/api/repositories/meta';
import { router } from '@/router';
import { useMetaStore } from '@/stores/meta';

/**
 * How often the idle welcome screen compares its version and start with the backend.
 * Most changes are noticed sooner, since every finished run returns to it.
 */
const VERSION_CHECK_INTERVAL_MS = 60_000;

/**
 * A reload right after a crash recovers from a one-off glitch.
 * A bug that returns right after the reload would loop quickly,
 * so once the reloads within the window reach the threshold, the next reload waits for the longer delay.
 */
const RELOAD_DELAY_MS = 2_000;
const RELOAD_BACKOFF_DELAY_MS = 30_000;
const RELOAD_BACKOFF_WINDOW_MS = 60_000;
const RELOAD_BACKOFF_THRESHOLD = 3;
const RELOAD_HISTORY_KEY = 'photopi:reloadHistory';
/**
 * How often a reload waits for an unreachable backend to answer again. Chromium in kiosk mode does not reload
 * its own error page, so a reload into a backend that is down would end on that page.
 */
const BACKEND_RETRY_DELAY_MS = 5_000;

/** The version of this build, set at build time like the version of the backend. */
const clientVersion = import.meta.env.VITE_APP_VERSION || 'dev';

/**
 * Browsers report a delayed ResizeObserver callback as an error event,
 * although they recover by the next frame, so it is no crash and must not reload the page.
 */
const RESIZE_OBSERVER_LOOP_MESSAGE = 'ResizeObserver loop';

let reloadScheduled = false;

/** recentReloadTimestamps returns the times of the reloads within the backoff window. */
function recentReloadTimestamps(now: number): number[] {
  try {
    const raw = sessionStorage.getItem(RELOAD_HISTORY_KEY);
    const timestamps: number[] = raw ? JSON.parse(raw) : [];
    const cutoff = now - RELOAD_BACKOFF_WINDOW_MS;
    return timestamps.filter((t) => t > cutoff);
  } catch {
    return [];
  }
}

/**
 * recordReload stores a reload at the given time and returns the delay before it.
 * Without session storage, the reload still happens, only without the backoff.
 */
export function recordReload(now: number): number {
  const timestamps = recentReloadTimestamps(now);
  timestamps.push(now);

  try {
    sessionStorage.setItem(RELOAD_HISTORY_KEY, JSON.stringify(timestamps));
  } catch {
    // The reload goes ahead without a history
  }

  return timestamps.length >= RELOAD_BACKOFF_THRESHOLD ? RELOAD_BACKOFF_DELAY_MS : RELOAD_DELAY_MS;
}

/**
 * reloadOnce reloads the page once, after the backend answers. The kiosk has no browser controls,
 * so a reload is the only way back from a stuck state. A reload that only applies a new backend passes onlyWhenIdle,
 * so it is dropped if a guest started a run in the meantime, and the next check on the welcome screen schedules it again.
 */
function reloadOnce(reason: string, detail?: unknown, onlyWhenIdle = false): void {
  if (reloadScheduled) return;
  reloadScheduled = true;

  console.warn(`[appRecovery] reloading due to: ${reason}`, detail);

  const delay = recordReload(Date.now());
  setTimeout(() => {
    if (onlyWhenIdle && !isIdle()) {
      reloadScheduled = false;
      return;
    }
    void waitForBackend().then(() => window.location.reload());
  }, delay);
}

/** isIdle reports whether the booth shows the welcome screen, where a reload discards nothing. */
function isIdle(): boolean {
  return router.currentRoute.value.name === 'welcome';
}

/** waitForBackend resolves once the backend answers, trying again until it does. */
async function waitForBackend(): Promise<void> {
  for (;;) {
    try {
      await metaRepository.get();
      return;
    } catch {
      await new Promise((resolve) => setTimeout(resolve, BACKEND_RETRY_DELAY_MS));
    }
  }
}

/**
 * checkBackendIfIdle reloads on the welcome screen, where a reload discards no run,
 * once the backend runs another version or was restarted since the page loaded,
 * since a restart applies a changed configuration. A page whose first metadata request failed reloads as well,
 * it would otherwise keep its fallbacks, such as the webcam as camera.
 */
async function checkBackendIfIdle(): Promise<void> {
  if (!isIdle()) return;

  try {
    const meta = await metaRepository.get();
    // The first request of the page may still run or have failed, loading it now settles both
    const metaStore = useMetaStore();
    const loaded = metaStore.meta ?? (await metaStore.load().catch(() => null));
    if (meta.version !== clientVersion) {
      reloadOnce(
        'version-mismatch',
        {
          backend: meta.version,
          client: clientVersion,
        },
        true,
      );
      return;
    }
    if (loaded?.instance !== meta.instance) {
      reloadOnce('backend-restarted', meta.instance, true);
    }
  } catch {
    // An unreachable backend is no crash and is handled by each screen
  }
}

/**
 * setupAppRecovery reloads the page on every uncaught error, and on a new or restarted backend
 * once the booth is idle again.
 */
export function setupAppRecovery(app: App): void {
  app.config.errorHandler = (err) => {
    console.error(err);
    reloadOnce('vue-error-handler', err);
  };

  useEventListener(window, 'error', (event: ErrorEvent) => {
    if (event.message.includes(RESIZE_OBSERVER_LOOP_MESSAGE)) return;
    reloadOnce('window-error', event.error ?? event);
  });
  useEventListener(window, 'unhandledrejection', (event: PromiseRejectionEvent) =>
    reloadOnce('unhandledrejection', event.reason),
  );

  router.afterEach((to) => {
    if (to.name === 'welcome') void checkBackendIfIdle();
  });

  setInterval(() => void checkBackendIfIdle(), VERSION_CHECK_INTERVAL_MS);
}
