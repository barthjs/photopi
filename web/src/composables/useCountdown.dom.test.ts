import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { withSetup } from '@/test/withSetup';

import { useCountdown } from '@/composables/useCountdown';

describe('useCountdown', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe('start', () => {
    it('starts at the given seconds', () => {
      const { result } = withSetup(() => useCountdown());

      result.start(3, () => {});

      expect(result.total.value).toBe(3);
      expect(result.remaining.value).toBe(3);
    });

    it('counts down once per second', () => {
      const { result } = withSetup(() => useCountdown());
      result.start(3, () => {});

      vi.advanceTimersByTime(1000);

      expect(result.remaining.value).toBe(2);
    });

    it('holds at zero before calling onDone', () => {
      const { result } = withSetup(() => useCountdown());
      const onDone = vi.fn<() => void>();
      result.start(3, onDone);

      vi.advanceTimersByTime(3000);

      expect(result.remaining.value).toBe(0);
      expect(onDone).not.toHaveBeenCalled();
    });

    it('calls onDone after the hold at zero', () => {
      const { result } = withSetup(() => useCountdown());
      const onDone = vi.fn<() => void>();
      result.start(3, onDone);

      vi.advanceTimersByTime(3200);

      expect(onDone).toHaveBeenCalledOnce();
    });

    it('calls onDone only once and stays at zero', () => {
      const { result } = withSetup(() => useCountdown());
      const onDone = vi.fn<() => void>();
      result.start(3, onDone);

      vi.advanceTimersByTime(8200);

      expect(onDone).toHaveBeenCalledOnce();
      expect(result.remaining.value).toBe(0);
    });

    it('restarts from the new duration', () => {
      const { result } = withSetup(() => useCountdown());
      result.start(3, () => {});
      vi.advanceTimersByTime(1000);

      result.start(5, () => {});
      vi.advanceTimersByTime(4000);

      expect(result.remaining.value).toBe(1);
    });

    it('calls only the onDone of the latest start', () => {
      const { result } = withSetup(() => useCountdown());
      const first = vi.fn<() => void>();
      const second = vi.fn<() => void>();
      result.start(3, first);
      vi.advanceTimersByTime(1000);

      result.start(5, second);
      vi.advanceTimersByTime(5200);

      expect(first).not.toHaveBeenCalled();
      expect(second).toHaveBeenCalledOnce();
    });
  });

  describe('stop', () => {
    it('cancels a running countdown and its zero hold', () => {
      const { result } = withSetup(() => useCountdown());
      const onDone = vi.fn<() => void>();
      result.start(1, onDone);
      vi.advanceTimersByTime(1000);

      result.stop();
      vi.advanceTimersByTime(1000);

      expect(onDone).not.toHaveBeenCalled();
    });

    it('stops when the component unmounts', () => {
      const { result, unmount } = withSetup(() => useCountdown());
      const onDone = vi.fn<() => void>();
      result.start(2, onDone);

      unmount();
      vi.advanceTimersByTime(5000);

      expect(onDone).not.toHaveBeenCalled();
      expect(result.remaining.value).toBe(2);
    });
  });
});
