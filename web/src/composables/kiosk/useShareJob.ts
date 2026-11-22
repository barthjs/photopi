import { onUnmounted, ref } from 'vue';

import type { ShareJobReason, ShareJobResponse } from '@/api/types/share';

/** How often the state of a running job is read. */
const POLL_INTERVAL_MS = 2_000;
/** The failed reads in a row that end the job for the guest, so a restart of the backend does not. */
const MAX_POLL_FAILURES = 3;

export type ShareJobPhase = 'idle' | 'pending' | 'done' | 'failed';

/**
 * Follows a background share job of the backend until it is done or failed.
 * The polling stops on unmount, also while a read is still in flight.
 */
export function useShareJob(read: () => Promise<ShareJobResponse>) {
  const phase = ref<ShareJobPhase>('idle');
  const reason = ref<ShareJobReason | null>(null);
  const url = ref('');

  let timer: ReturnType<typeof setTimeout> | null = null;
  let failures = 0;
  // Increased on every track and stop, so a read that returns after them is ignored
  let generation = 0;
  let unmounted = false;

  /** Starts following a job the caller just started at the backend. */
  function track(): void {
    stop();
    if (unmounted) return;

    phase.value = 'pending';
    reason.value = null;
    url.value = '';
    failures = 0;
    schedule(generation);
  }

  /** Ends the job as failed, for a job that could not be started at all. */
  function fail(failedReason: ShareJobReason): void {
    stop();
    phase.value = 'failed';
    reason.value = failedReason;
  }

  /** Ends the polling without changing the phase. */
  function stop(): void {
    generation++;
    if (timer !== null) {
      clearTimeout(timer);
      timer = null;
    }
  }

  /** Reads the state of the job after the poll interval. */
  function schedule(current: number): void {
    timer = setTimeout(() => void poll(current), POLL_INTERVAL_MS);
  }

  /** Reads the state once and schedules the next read while the job is pending. */
  async function poll(current: number): Promise<void> {
    let state: ShareJobResponse;
    try {
      state = await read();
    } catch {
      if (current !== generation) return;
      failures++;
      if (failures >= MAX_POLL_FAILURES) {
        fail('failed');
        return;
      }
      schedule(current);
      return;
    }

    if (current !== generation) return;
    failures = 0;

    if (state.status === 'pending') {
      schedule(current);
      return;
    }

    timer = null;
    phase.value = state.status;
    reason.value = state.reason ?? null;
    url.value = state.url ?? '';
  }

  onUnmounted(() => {
    unmounted = true;
    stop();
  });

  return {
    phase,
    reason,
    url,
    track,
    fail,
    stop,
  };
}
