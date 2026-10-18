import { describe, expect, it } from 'vitest';

import { publicShareRepository } from '@/api/repositories/publicShare';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('publicShareRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('get', () => {
    it('returns the shared run under its encoded token', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, {
          imageUrls: ['/media/captures/a.jpg?signature=x'],
        }),
      );

      const result = await publicShareRepository.get('a/b');

      expect(result).toEqual({
        imageUrls: ['/media/captures/a.jpg?signature=x'],
      });
      expect(lastRequest().url).toBe('/api/share/public/a%2Fb');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe('zipUrl', () => {
    it('returns the address of the archive', () => {
      expect(publicShareRepository.zipUrl('abc')).toBe('/api/share/public/abc/zip');
    });
  });
});
