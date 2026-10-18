import { describe, expect, it } from 'vitest';

import { captureRepository } from '@/api/repositories/capture';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('captureRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('create', () => {
    it('stores a capture without a body for a source that takes its own photo', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(201, {
          run: 3,
          imageIndex: 1,
          imagesPerRun: 4,
        }),
      );

      const result = await captureRepository.create(3);

      expect(result).toEqual({
        run: 3,
        imageIndex: 1,
        imagesPerRun: 4,
      });
      expect(lastRequest().url).toBe('/api/camera/runs/3/captures');
      expect(lastRequest().init?.method).toBe('POST');
      expect(lastRequest().init?.body).toBeUndefined();
    });

    it('uploads the still taken by the browser', async () => {
      fetchMock.mockResolvedValue(jsonResponse(201, {}));
      const blob = new Blob(['image'], {
        type: 'image/jpeg',
      });

      await captureRepository.create(3, blob);

      expect(lastRequest().url).toBe('/api/camera/runs/3/captures');
      expect(lastRequest().init?.method).toBe('POST');
      const body = lastRequest().init?.body;
      expect(body).toBeInstanceOf(FormData);
      expect((body as FormData).get('file')).toBeInstanceOf(Blob);
    });
  });

  describe('previewUrl', () => {
    it('returns the address of the screen version of a capture', () => {
      expect(captureRepository.previewUrl(3, 1)).toBe('/api/camera/runs/3/captures/1/preview');
    });
  });
});
