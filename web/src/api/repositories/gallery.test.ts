import { describe, expect, it } from 'vitest';

import { galleryRepository } from '@/api/repositories/gallery';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('galleryRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('list', () => {
    it('returns a page of runs', async () => {
      const page = {
        data: [],
        meta: {
          totalCount: 0,
          limit: 24,
          offset: 48,
        },
      };
      fetchMock.mockResolvedValue(jsonResponse(200, page));

      const result = await galleryRepository.list({
        filters: {
          event: 'summer-party',
        },
        limit: 24,
        offset: 48,
      });

      expect(result).toEqual(page);
      expect(lastRequest().url).toBe(
        '/api/gallery?filter%5Bevent%5D=summer-party&page%5Blimit%5D=24&page%5Boffset%5D=48',
      );
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe('delete', () => {
    it('removes the run', async () => {
      fetchMock.mockResolvedValue(jsonResponse(204, undefined));

      await galleryRepository.delete(5);

      expect(lastRequest().url).toBe('/api/gallery/5');
      expect(lastRequest().init?.method).toBe('DELETE');
    });
  });

  describe('listEvents', () => {
    it('returns the events', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, [
          {
            event: 'summer-party',
          },
        ]),
      );

      const result = await galleryRepository.listEvents();

      expect(result).toEqual([
        {
          event: 'summer-party',
        },
      ]);
      expect(lastRequest().url).toBe('/api/gallery/events');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe('captureUrl', () => {
    it('returns the address of a capture', () => {
      expect(galleryRepository.captureUrl(5, 2)).toBe('/api/gallery/5/captures/2');
    });
  });

  describe('originalUrl', () => {
    it('returns the address of the original of a capture', () => {
      expect(galleryRepository.originalUrl(5, 2)).toBe('/api/gallery/5/originals/2');
    });
  });

  describe('zipUrl', () => {
    it('returns the address of the archive of a run', () => {
      expect(galleryRepository.zipUrl(5)).toBe('/api/gallery/5/zip');
    });

    it('returns the address of the archive with the originals of a run', () => {
      expect(galleryRepository.zipUrl(5, true)).toBe('/api/gallery/5/zip?originals=true');
    });
  });

  describe('eventZipUrl', () => {
    it('returns the address of the archive of an event', () => {
      expect(galleryRepository.eventZipUrl('summer-party')).toBe(
        '/api/gallery/events/summer-party/zip',
      );
    });

    it('returns the address of the archive with the originals of an event', () => {
      expect(galleryRepository.eventZipUrl('summer-party', true)).toBe(
        '/api/gallery/events/summer-party/zip?originals=true',
      );
    });
  });
});
