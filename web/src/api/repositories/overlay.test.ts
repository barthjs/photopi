import { describe, expect, it } from 'vitest';

import { overlayRepository } from '@/api/repositories/overlay';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('overlayRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('list', () => {
    it('returns the overlays', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, [
          {
            name: 'overlay.png',
            path: 'overlays/overlay.png',
          },
        ]),
      );

      const result = await overlayRepository.list();

      expect(result).toEqual([
        {
          name: 'overlay.png',
          path: 'overlays/overlay.png',
        },
      ]);
      expect(lastRequest().url).toBe('/api/overlays');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe('create', () => {
    it('uploads the overlay as a file', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(201, {
          name: 'overlay.png',
          path: 'overlays/overlay.png',
        }),
      );
      const file = new File(['png'], 'overlay.png', {
        type: 'image/png',
      });

      const result = await overlayRepository.create(file);

      expect(result).toEqual({
        name: 'overlay.png',
        path: 'overlays/overlay.png',
      });
      expect(lastRequest().url).toBe('/api/overlays');
      expect(lastRequest().init?.method).toBe('POST');
      const body = lastRequest().init?.body;
      expect(body).toBeInstanceOf(FormData);
      expect((body as FormData).get('file')).toBeInstanceOf(File);
    });
  });

  describe('delete', () => {
    it('removes the overlay under its encoded name', async () => {
      fetchMock.mockResolvedValue(jsonResponse(204, undefined));

      await overlayRepository.delete('an overlay.png');

      expect(lastRequest().url).toBe('/api/overlays/an%20overlay.png');
      expect(lastRequest().init?.method).toBe('DELETE');
    });
  });

  describe('downloadUrl', () => {
    it('returns the download address under the encoded name', () => {
      const result = overlayRepository.downloadUrl('an overlay.png');

      expect(result).toBe('/api/overlays/an%20overlay.png');
    });
  });
});
