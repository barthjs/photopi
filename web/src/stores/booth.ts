import { defineStore } from 'pinia';
import { ref } from 'vue';

import { runRepository } from '@/api/repositories/run';
import { screenPixels } from '@/lib/browser/screen';

/** Holds the run of the current guest across the screens of the booth. */
export const useBoothStore = defineStore('booth', () => {
  const run = ref<number | null>(null);
  const imagesPerRun = ref<number | null>(null);
  const capturedCount = ref(0);

  /** Starts a run at the backend. */
  async function startRun(): Promise<void> {
    const result = await runRepository.create(screenPixels(window));
    run.value = result.run;
    imagesPerRun.value = result.imagesPerRun;
    capturedCount.value = 0;
  }

  /** Counts a taken photo. */
  function recordCapture(): void {
    capturedCount.value += 1;
  }

  /** Forgets the run. */
  function reset(): void {
    run.value = null;
    imagesPerRun.value = null;
    capturedCount.value = 0;
  }

  /**
   * Discards the run and its captures at the backend.
   * It never fails, since the booth has to return to the welcome screen either way:
   * a run the backend could not discard is discarded by the backend once the booth starts the next one.
   */
  async function abort(): Promise<void> {
    try {
      if (run.value !== null) {
        await runRepository.abort(run.value);
      }
    } catch (err) {
      console.warn('[booth] discarding the run failed', err);
    } finally {
      reset();
    }
  }

  return {
    run,
    imagesPerRun,
    capturedCount,
    startRun,
    recordCapture,
    reset,
    abort,
  };
});
