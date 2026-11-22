import { onUnmounted } from 'vue';
import { useEventListener } from '@vueuse/core';

/**
 * Calls onTimeout once no pointer or keyboard input happened for the duration last passed to arm.
 * armFixed starts a delay that input does not extend, and clear stops the timer.
 */
export function useInactivityTimer(onTimeout: () => void) {
  let timer: ReturnType<typeof setTimeout> | null = null;
  let armedMs = 0;

  /** Cancels the pending timeout. */
  function clearTimer(): void {
    if (timer !== null) {
      clearTimeout(timer);
      timer = null;
    }
  }

  /** Starts the timeout with the given duration, which every input restarts. */
  function arm(timeoutMs: number): void {
    armedMs = timeoutMs;
    clearTimer();
    timer = setTimeout(onTimeout, timeoutMs);
  }

  /** Starts the timeout with the given duration, which input does not restart. */
  function armFixed(timeoutMs: number): void {
    armedMs = 0;
    clearTimer();
    timer = setTimeout(onTimeout, timeoutMs);
  }

  /** Stops watching for inactivity. */
  function clear(): void {
    armedMs = 0;
    clearTimer();
  }

  /** Restarts the timeout after an input. */
  function handleActivity(): void {
    if (armedMs > 0) arm(armedMs);
  }

  useEventListener(window, 'pointerdown', handleActivity);
  useEventListener(window, 'keydown', handleActivity);

  onUnmounted(clear);

  return {
    arm,
    armFixed,
    clear,
  };
}
