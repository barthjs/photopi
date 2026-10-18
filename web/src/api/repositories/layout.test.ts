import { describe, expect, it } from 'vitest';

import { layoutRepository } from '@/api/repositories/layout';
import type { LayoutSaveRequest } from '@/api/types';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('layoutRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('list', () => {
    it('returns the layouts', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, [
          {
            id: 'layout-a',
          },
        ]),
      );

      const result = await layoutRepository.list();

      expect(result).toEqual([
        {
          id: 'layout-a',
        },
      ]);
      expect(lastRequest().url).toBe('/api/print/layouts');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe('get', () => {
    it('returns the layout under its encoded ID', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, {
          id: 'layout a/b',
        }),
      );

      const result = await layoutRepository.get('layout a/b');

      expect(result).toEqual({
        id: 'layout a/b',
      });
      expect(lastRequest().url).toBe('/api/print/layouts/layout%20a%2Fb');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe('save', () => {
    it('stores the layout under its encoded ID', async () => {
      const payload = {
        name: 'Layout',
      } as unknown as LayoutSaveRequest;
      fetchMock.mockResolvedValue(jsonResponse(200, payload));

      const result = await layoutRepository.save('layout a/b', payload);

      expect(result).toEqual(payload);
      expect(lastRequest().url).toBe('/api/print/layouts/layout%20a%2Fb');
      expect(lastRequest().init?.method).toBe('PUT');
      expect(lastRequest().init?.body).toBe(JSON.stringify(payload));
    });
  });

  describe('delete', () => {
    it('removes the layout under its encoded ID', async () => {
      fetchMock.mockResolvedValue(jsonResponse(204, undefined));

      await layoutRepository.delete('layout a/b');

      expect(lastRequest().url).toBe('/api/print/layouts/layout%20a%2Fb');
      expect(lastRequest().init?.method).toBe('DELETE');
    });
  });
});
