import { nextTick, ref } from 'vue';
import { describe, expect, it } from 'vitest';

import { useAspectRatioLock } from '@/composables/useAspectRatioLock';

/** setup creates an unlocked ratio lock for a width and height held in refs. */
function setup(initialWidth: number, initialHeight: number) {
  const width = ref(initialWidth);
  const height = ref(initialHeight);
  const lock = useAspectRatioLock(
    {
      get: () => width.value,
      set: (value) => (width.value = value),
    },
    {
      get: () => height.value,
      set: (value) => (height.value = value),
    },
  );
  return {
    width,
    height,
    lock,
  };
}

describe('useAspectRatioLock', () => {
  describe('lock', () => {
    it('changes the height with the width at the current ratio', async () => {
      const { width, height, lock } = setup(1920, 1080);
      lock.lock();

      width.value = 960;
      await nextTick();

      expect(height.value).toBe(540);
    });

    it('keeps dimensions changed together when it locks before the fields follow', async () => {
      const { width, height, lock } = setup(1920, 1080);
      lock.lock();
      width.value = 1000;
      height.value = 1000;

      lock.lock();
      await nextTick();

      expect([width.value, height.value]).toEqual([1000, 1000]);
    });
  });

  describe('toggle', () => {
    it('changes the height with the width while locked', async () => {
      const { width, height, lock } = setup(1920, 1080);
      lock.toggle();

      width.value = 1280;
      await nextTick();

      expect(height.value).toBe(720);
    });

    it('changes the width with the height while locked', async () => {
      const { width, height, lock } = setup(1920, 1080);
      lock.toggle();
      width.value = 1280;
      await nextTick();

      height.value = 1080;
      await nextTick();

      expect(width.value).toBe(1920);
    });

    it('captures the current ratio on every lock', async () => {
      const { width, height, lock } = setup(1920, 1080);
      lock.toggle();
      lock.toggle();
      width.value = 1000;
      height.value = 1000;
      await nextTick();
      lock.toggle();

      width.value = 500;
      await nextTick();

      expect(height.value).toBe(500);
    });

    it('leaves the fields independent until locked', async () => {
      const { width, height, lock } = setup(1920, 1080);

      width.value = 1000;
      await nextTick();

      expect(lock.locked.value).toBe(false);
      expect(height.value).toBe(1080);
    });

    it('stops following after unlocking', async () => {
      const { width, height, lock } = setup(1920, 1080);
      lock.toggle();
      lock.toggle();

      width.value = 100;
      await nextTick();

      expect(lock.locked.value).toBe(false);
      expect(height.value).toBe(1080);
    });

    it('never sets a dimension below 1', async () => {
      const { width, height, lock } = setup(1920, 1080);
      lock.toggle();

      width.value = 0;
      await nextTick();

      expect(height.value).toBe(1);
    });
  });

  describe('label', () => {
    it('labels the ratio in lowest terms', () => {
      const { lock } = setup(1920, 1080);

      expect(lock.label.value).toBe('16:9');
    });

    it('follows changed dimensions', async () => {
      const { width, height, lock } = setup(1920, 1080);

      width.value = 1000;
      height.value = 1000;
      await nextTick();

      expect(lock.label.value).toBe('1:1');
    });

    it.each([
      {
        missing: 'width',
        width: 0,
        height: 1080,
      },
      {
        missing: 'height',
        width: 1920,
        height: 0,
      },
    ])('shows a placeholder while the $missing is missing', ({ width, height }) => {
      const { lock } = setup(width, height);

      expect(lock.label.value).toBe('–');
    });
  });
});
