import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { runRepository } from '@/api/repositories/run';

import { useBoothStore } from '@/stores/booth';

vi.mock('@/api/repositories/run', () => ({
  runRepository: {
    create: vi.fn<typeof runRepository.create>(),
    abort: vi.fn<typeof runRepository.abort>(),
  },
}));

describe('useBoothStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.mocked(runRepository.create).mockReset();
    vi.mocked(runRepository.abort).mockReset();
  });

  describe('startRun', () => {
    it('starts a run with the number and size the backend assigned', async () => {
      vi.mocked(runRepository.create).mockResolvedValue({
        run: 7,
        imagesPerRun: 4,
      });
      const store = useBoothStore();
      store.recordCapture();

      await store.startRun();

      expect(store.run).toBe(7);
      expect(store.imagesPerRun).toBe(4);
      expect(store.capturedCount).toBe(0);
    });
  });

  describe('recordCapture', () => {
    it('counts a taken photo', () => {
      const store = useBoothStore();
      store.run = 3;
      store.imagesPerRun = 4;
      store.recordCapture();

      store.recordCapture();

      expect(store.capturedCount).toBe(2);
    });
  });

  describe('reset', () => {
    it('forgets the run', () => {
      const store = useBoothStore();
      store.run = 3;
      store.imagesPerRun = 4;
      store.recordCapture();

      store.reset();

      expect(store.run).toBeNull();
      expect(store.imagesPerRun).toBeNull();
      expect(store.capturedCount).toBe(0);
    });
  });

  describe('abort', () => {
    it('aborts the run at the backend and resets', async () => {
      vi.mocked(runRepository.abort).mockResolvedValue(undefined);
      const store = useBoothStore();
      store.run = 5;
      store.recordCapture();

      await store.abort();

      expect(runRepository.abort).toHaveBeenCalledWith(5);
      expect(store.run).toBeNull();
      expect(store.capturedCount).toBe(0);
    });

    it('does not call the backend without a run', async () => {
      const store = useBoothStore();

      await store.abort();

      expect(runRepository.abort).not.toHaveBeenCalled();
    });

    it('resets without failing when aborting at the backend fails', async () => {
      vi.mocked(runRepository.abort).mockRejectedValue(new Error('offline'));
      vi.spyOn(console, 'warn').mockImplementation(() => {});
      const store = useBoothStore();
      store.run = 5;

      const aborted = store.abort();

      await expect(aborted).resolves.toBeUndefined();
      expect(store.run).toBeNull();
    });
  });
});
