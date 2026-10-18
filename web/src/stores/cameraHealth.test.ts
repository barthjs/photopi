import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useFetchMock } from '@/test/fetchMock';

import { useCameraHealthStore } from '@/stores/cameraHealth';

/** statusResponse returns a readiness response without a body. */
function statusResponse(status: number): Response {
  return new Response(null, {
    status,
  });
}

/** degradedResponse returns a successful readiness response with the given state of the camera check. */
function degradedResponse(camera: string): Response {
  return new Response(
    JSON.stringify({
      status: 'degraded',
      checks: {
        camera,
      },
    }),
    {
      status: 200,
    },
  );
}

describe('useCameraHealthStore', () => {
  const { fetchMock } = useFetchMock();

  beforeEach(() => {
    setActivePinia(createPinia());
    vi.useFakeTimers();
  });

  describe('available', () => {
    it('assumes the camera is available until the first check', () => {
      const store = useCameraHealthStore();

      expect(store.available).toBe(true);
    });
  });

  describe('start', () => {
    it('checks the readiness probe right away', async () => {
      fetchMock.mockResolvedValueOnce(statusResponse(503));
      const store = useCameraHealthStore();

      store.start();

      await vi.waitFor(() => expect(store.available).toBe(false));
      expect(fetchMock).toHaveBeenCalledWith('/readyz');
    });

    it('checks the readiness probe again after the interval', async () => {
      fetchMock.mockResolvedValueOnce(statusResponse(503));
      const store = useCameraHealthStore();
      store.start();
      await vi.waitFor(() => expect(store.available).toBe(false));
      fetchMock.mockResolvedValueOnce(statusResponse(200));

      await vi.advanceTimersByTimeAsync(5000);

      expect(store.available).toBe(true);
    });

    it('reads an unavailable camera from the check of a degraded response', async () => {
      fetchMock.mockResolvedValueOnce(degradedResponse('unreachable'));
      const store = useCameraHealthStore();

      store.start();

      await vi.waitFor(() => expect(store.available).toBe(false));
    });

    it('reads an available camera from the check of a degraded response', async () => {
      fetchMock.mockResolvedValueOnce(degradedResponse('unreachable'));
      const store = useCameraHealthStore();
      store.start();
      await vi.waitFor(() => expect(store.available).toBe(false));
      fetchMock.mockResolvedValueOnce(degradedResponse('ok'));

      await vi.advanceTimersByTimeAsync(5000);

      expect(store.available).toBe(true);
    });

    it('keeps the last state on a network error', async () => {
      fetchMock.mockResolvedValueOnce(statusResponse(503));
      const store = useCameraHealthStore();
      store.start();
      await vi.waitFor(() => expect(store.available).toBe(false));
      fetchMock.mockRejectedValueOnce(new TypeError('Failed to fetch'));

      await vi.advanceTimersByTimeAsync(5000);

      expect(store.available).toBe(false);
    });

    it('polls only once however often it is started', async () => {
      fetchMock.mockResolvedValue(statusResponse(200));
      const store = useCameraHealthStore();
      store.start();

      store.start();
      await vi.advanceTimersByTimeAsync(5000);

      expect(fetchMock).toHaveBeenCalledTimes(2);
    });
  });
});
