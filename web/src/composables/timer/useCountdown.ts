import { onUnmounted, ref } from 'vue';

/**
 * How long the countdown holds at zero before onDone fires,
 * so the browser paints the finished state before whatever onDone shows next.
 */
const ZERO_HOLD_MS = 200;

/** Counts whole seconds down to zero and calls onDone after a short hold at zero. */
export function useCountdown() {
  const remaining = ref(0);
  const total = ref(0);
  let interval: ReturnType<typeof setInterval> | null = null;
  let holdTimeout: ReturnType<typeof setTimeout> | null = null;

  /** Cancels a running countdown. */
  function stop() {
    if (interval !== null) {
      clearInterval(interval);
      interval = null;
    }
    if (holdTimeout !== null) {
      clearTimeout(holdTimeout);
      holdTimeout = null;
    }
  }

  /** Counts down the given seconds and calls onDone at zero. */
  function start(seconds: number, onDone: () => void) {
    stop();
    total.value = seconds;
    remaining.value = seconds;

    interval = setInterval(() => {
      remaining.value -= 1;
      if (remaining.value <= 0) {
        stop();
        holdTimeout = setTimeout(onDone, ZERO_HOLD_MS);
      }
    }, 1000);
  }

  onUnmounted(stop);

  return {
    remaining,
    total,
    start,
    stop,
  };
}
