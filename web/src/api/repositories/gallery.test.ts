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

  describe('listDays', () => {
    it('returns the runs per day of the event', async () => {
      const days = {
        days: [
          {
            day: '2026-11-17',
            count: 3,
          },
        ],
      };
      fetchMock.mockResolvedValue(jsonResponse(200, days));

      const result = await galleryRepository.listDays({
        event: 'summer-party',
      });

      expect(result).toEqual(days);
      expect(lastRequest().url).toBe('/api/gallery/days?filter%5Bevent%5D=summer-party');
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

  describe('bulk', () => {
    it('deletes the runs', async () => {
      const response = {
        succeeded: 2,
        failed: 0,
        skipped: 0,
        failedIds: [],
      };
      fetchMock.mockResolvedValue(jsonResponse(200, response));

      const result = await galleryRepository.bulk({
        action: 'delete',
        ids: ['12', '13'],
      });

      expect(result).toEqual(response);
      expect(lastRequest().url).toBe('/api/gallery/bulk');
      expect(lastRequest().init?.method).toBe('POST');
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

  describe('thumbnailUrl', () => {
    it('returns the address of the thumbnail of a run', () => {
      expect(galleryRepository.thumbnailUrl(12)).toBe('/api/gallery/12/thumbnail');
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

  describe('selectionZipUrl', () => {
    it('returns the address of the archive of the runs', () => {
      expect(galleryRepository.selectionZipUrl([12, 13])).toBe(
        '/api/gallery/zip?filter%5Brun%5D=12%2C13',
      );
    });

    it('returns the address of the archive of their originals', () => {
      expect(galleryRepository.selectionZipUrl([12], true)).toBe(
        '/api/gallery/zip?filter%5Brun%5D=12&originals=true',
      );
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
