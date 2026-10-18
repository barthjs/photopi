import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { withSetup } from '@/test/withSetup';

import { useInactivityTimer } from '@/composables/useInactivityTimer';

const activities = [
  ['a pointer press', () => new PointerEvent('pointerdown')],
  [
    'a key press',
    () =>
      new KeyboardEvent('keydown', {
        key: 'a',
      }),
  ],
] as const;

describe('useInactivityTimer', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe('arm', () => {
    it('times out after the armed duration without activity', () => {
      const onTimeout = vi.fn<() => void>();
      const { result } = withSetup(() => useInactivityTimer(onTimeout));
      result.arm(1000);

      vi.advanceTimersByTime(1000);

      expect(onTimeout).toHaveBeenCalledOnce();
    });

    it('waits for the full armed duration', () => {
      const onTimeout = vi.fn<() => void>();
      const { result } = withSetup(() => useInactivityTimer(onTimeout));
      result.arm(1000);

      vi.advanceTimersByTime(999);

      expect(onTimeout).not.toHaveBeenCalled();
    });

    it.each(activities)('restarts the wait on %s', (_, event) => {
      const onTimeout = vi.fn<() => void>();
      const { result } = withSetup(() => useInactivityTimer(onTimeout));
      result.arm(1000);
      vi.advanceTimersByTime(800);

      window.dispatchEvent(event());
      vi.advanceTimersByTime(800);

      expect(onTimeout).not.toHaveBeenCalled();
    });

    it.each(activities)('times out after the wait restarted by %s', (_, event) => {
      const onTimeout = vi.fn<() => void>();
      const { result } = withSetup(() => useInactivityTimer(onTimeout));
      result.arm(1000);
      vi.advanceTimersByTime(800);

      window.dispatchEvent(event());
      vi.advanceTimersByTime(1000);

      expect(onTimeout).toHaveBeenCalledOnce();
    });

    it('uses the duration passed last', () => {
      const onTimeout = vi.fn<() => void>();
      const { result } = withSetup(() => useInactivityTimer(onTimeout));
      result.arm(1000);

      result.arm(5000);
      vi.advanceTimersByTime(4000);
      window.dispatchEvent(new PointerEvent('pointerdown'));
      vi.advanceTimersByTime(4999);

      expect(onTimeout).not.toHaveBeenCalled();
    });
  });

  describe('clear', () => {
    it('ignores activity once cleared', () => {
      const onTimeout = vi.fn<() => void>();
      const { result } = withSetup(() => useInactivityTimer(onTimeout));
      result.arm(1000);

      result.clear();
      window.dispatchEvent(new PointerEvent('pointerdown'));
      vi.advanceTimersByTime(10_000);

      expect(onTimeout).not.toHaveBeenCalled();
    });

    it('stops listening and disarms when the component unmounts', () => {
      const onTimeout = vi.fn<() => void>();
      const { result, unmount } = withSetup(() => useInactivityTimer(onTimeout));
      result.arm(1000);

      unmount();
      window.dispatchEvent(new PointerEvent('pointerdown'));
      vi.advanceTimersByTime(10_000);

      expect(onTimeout).not.toHaveBeenCalled();
    });
  });
});
