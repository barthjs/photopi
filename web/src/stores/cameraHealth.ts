import { defineStore } from 'pinia';
import { ref } from 'vue';

/** How often the readiness of the camera is checked, so a guest cannot start a run without it. */
const POLL_INTERVAL_MS = 5_000;

// The readiness response names the state of every dependency. The camera is optional,
// so a failing camera only shows in its check, while the response stays successful.
type ReadinessResponse = {
  checks?: Record<string, string>;
};

/**
 * cameraAvailable reads the camera check of a readiness response. Without a readable check, the HTTP status decides,
 * as a failing required dependency answers 503.
 */
async function cameraAvailable(response: Response): Promise<boolean> {
  try {
    // eslint-disable-next-line @typescript-eslint/no-unsafe-type-assertion -- the readiness endpoint answers with this shape
    const body = (await response.json()) as ReadinessResponse;
    const camera = body.checks?.camera;
    if (camera !== undefined) return camera === 'ok';
  } catch {
    // Without a JSON body, the status decides
  }
  return response.ok;
}

/** useCameraHealthStore polls and holds whether the camera is available. */
export const useCameraHealthStore = defineStore('cameraHealth', () => {
  // Optimistic until the first poll, so the welcome screen does not flash an unavailable camera
  const available = ref(true);

  let timer: ReturnType<typeof setInterval> | null = null;

  /** poll reads whether the camera is available. */
  async function poll(): Promise<void> {
    try {
      // The readiness endpoint lies outside the API and needs none of the behavior of apiFetch
      const response = await fetch('/readyz');
      available.value = await cameraAvailable(response);
    } catch {
      // A network error does not prove a failing camera, so the last state is kept
    }
  }

  /** start polls whether the camera is available at a fixed interval. */
  function start(): void {
    if (timer !== null) return;
    void poll();
    timer = setInterval(() => void poll(), POLL_INTERVAL_MS);
  }

  return {
    available,
    start,
  };
});
