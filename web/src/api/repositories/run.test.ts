import { describe, expect, it } from 'vitest';

import { runRepository } from '@/api/repositories/run';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('runRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('create', () => {
    it('starts a run', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(201, {
          run: 3,
          imagesPerRun: 4,
        }),
      );

      const result = await runRepository.create();

      expect(result).toEqual({
        run: 3,
        imagesPerRun: 4,
      });
      expect(lastRequest().url).toBe('/api/camera/runs');
      expect(lastRequest().init?.method).toBe('POST');
    });
  });

  describe('abort', () => {
    it('deletes the run to abort it', async () => {
      fetchMock.mockResolvedValue(jsonResponse(204, undefined));

      await runRepository.abort(3);

      expect(lastRequest().url).toBe('/api/camera/runs/3');
      expect(lastRequest().init?.method).toBe('DELETE');
    });
  });
});
