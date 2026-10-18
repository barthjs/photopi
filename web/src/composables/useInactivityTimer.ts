import { onUnmounted } from 'vue';
import { useEventListener } from '@vueuse/core';

/**
 * useInactivityTimer calls onTimeout once no pointer or keyboard input happened for the duration last passed to arm.
 * clear stops the timer for a state with a fixed delay that input must not extend.
 */
export function useInactivityTimer(onTimeout: () => void) {
  let timer: ReturnType<typeof setTimeout> | null = null;
  let armedMs = 0;

  /** clearTimer cancels the pending timeout. */
  function clearTimer(): void {
    if (timer !== null) {
      clearTimeout(timer);
      timer = null;
    }
  }

  /** arm starts the timeout with the given duration, which every guest activity restarts. */
  function arm(timeoutMs: number): void {
    armedMs = timeoutMs;
    clearTimer();
    timer = setTimeout(onTimeout, timeoutMs);
  }

  /** clear stops watching for inactivity. */
  function clear(): void {
    armedMs = 0;
    clearTimer();
  }

  /** handleActivity restarts the timeout after a guest activity. */
  function handleActivity(): void {
    if (armedMs > 0) arm(armedMs);
  }

  useEventListener(window, 'pointerdown', handleActivity);
  useEventListener(window, 'keydown', handleActivity);

  onUnmounted(clear);

  return {
    arm,
    clear,
  };
}
