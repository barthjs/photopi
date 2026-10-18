import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { ShareJobResponse } from '@/api/types';
import { withSetup } from '@/test/withSetup';

import { useShareJob } from '@/composables/useShareJob';

/** pendingThenDone returns a read that reports the job as pending once and then as done with a link. */
function pendingThenDone() {
  return vi
    .fn<() => Promise<ShareJobResponse>>()
    .mockResolvedValueOnce({
      status: 'pending',
    })
    .mockResolvedValueOnce({
      status: 'done',
      url: 'https://example.com/s',
    });
}

describe('useShareJob', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe('track', () => {
    it('marks the job as pending', () => {
      const { result } = withSetup(() => useShareJob(pendingThenDone()));

      result.track();

      expect(result.phase.value).toBe('pending');
    });

    it('keeps polling while the job is pending', async () => {
      const { result } = withSetup(() => useShareJob(pendingThenDone()));
      result.track();

      await vi.advanceTimersByTimeAsync(2_000);

      expect(result.phase.value).toBe('pending');
    });

    it('keeps the link of a finished job', async () => {
      const { result } = withSetup(() => useShareJob(pendingThenDone()));
      result.track();

      await vi.advanceTimersByTimeAsync(4_000);

      expect(result.phase.value).toBe('done');
      expect(result.url.value).toBe('https://example.com/s');
    });

    it('reports the reason of a failed job', async () => {
      const read = vi.fn<() => Promise<ShareJobResponse>>().mockResolvedValue({
        status: 'failed',
        reason: 'unreachable',
      });
      const { result } = withSetup(() => useShareJob(read));
      result.track();

      await vi.advanceTimersByTimeAsync(2_000);

      expect(result.phase.value).toBe('failed');
      expect(result.reason.value).toBe('unreachable');
    });

    it('stops polling once the job is finished', async () => {
      const read = pendingThenDone();
      const { result } = withSetup(() => useShareJob(read));
      result.track();

      await vi.advanceTimersByTimeAsync(14_000);

      expect(read).toHaveBeenCalledTimes(2);
    });

    it('tolerates a single failed read', async () => {
      const read = vi
        .fn<() => Promise<ShareJobResponse>>()
        .mockRejectedValueOnce(new Error('offline'))
        .mockResolvedValueOnce({
          status: 'done',
        });
      const { result } = withSetup(() => useShareJob(read));
      result.track();

      await vi.advanceTimersByTimeAsync(4_000);

      expect(result.phase.value).toBe('done');
    });

    it('fails after repeated failed reads', async () => {
      const read = vi.fn<() => Promise<ShareJobResponse>>().mockRejectedValue(new Error('offline'));
      const { result } = withSetup(() => useShareJob(read));
      result.track();

      await vi.advanceTimersByTimeAsync(6_000);

      expect(result.phase.value).toBe('failed');
      expect(result.reason.value).toBe('failed');
    });

    it('stops polling after repeated failed reads', async () => {
      const read = vi.fn<() => Promise<ShareJobResponse>>().mockRejectedValue(new Error('offline'));
      const { result } = withSetup(() => useShareJob(read));
      result.track();

      await vi.advanceTimersByTimeAsync(16_000);

      expect(read).toHaveBeenCalledTimes(3);
    });

    it('stops polling on unmount, also with a read in flight', async () => {
      let resolveRead: (state: ShareJobResponse) => void = () => {};
      const read = vi.fn<() => Promise<ShareJobResponse>>().mockImplementation(
        () =>
          new Promise((resolve) => {
            resolveRead = resolve;
          }),
      );
      const { result, unmount } = withSetup(() => useShareJob(read));
      result.track();
      await vi.advanceTimersByTimeAsync(2_000);

      unmount();
      resolveRead({
        status: 'pending',
      });
      await vi.advanceTimersByTimeAsync(10_000);

      expect(read).toHaveBeenCalledOnce();
    });

    it('does not start after unmount', async () => {
      const read = vi.fn<() => Promise<ShareJobResponse>>();
      const { result, unmount } = withSetup(() => useShareJob(read));
      unmount();

      result.track();
      await vi.advanceTimersByTimeAsync(10_000);

      expect(read).not.toHaveBeenCalled();
      expect(result.phase.value).toBe('idle');
    });
  });
});
