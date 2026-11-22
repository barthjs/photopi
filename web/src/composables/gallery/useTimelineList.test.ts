import { describe, expect, it, vi } from 'vitest';

import type { PaginatedResponse, PaginationParams } from '@/api/client/list';
import { useTimelineList } from '@/composables/gallery/useTimelineList';

vi.mock('@/composables/form/useApiError', () => ({
  useApiError: () => ({
    messageFor: (err: unknown) => (err instanceof Error ? err.message : String(err)),
  }),
}));

/** Returns the page of the numbers from the offset, of a list of the total. */
function pageOf(page: PaginationParams, total: number): PaginatedResponse<number> {
  const limit = page.limit ?? total;
  const offset = page.offset ?? 0;
  const data: number[] = [];
  for (let index = offset; index < Math.min(offset + limit, total); index += 1) data.push(index);
  return {
    data,
    meta: {
      totalCount: total,
      limit,
      offset,
    },
  };
}

describe('useTimelineList', () => {
  describe('reload', () => {
    it('loads the first page and the periods', async () => {
      const periods = [
        {
          period: '2026-11',
          count: 5,
        },
      ];
      const timeline = useTimelineList<number>({
        pageSize: 2,
        fetch: async (page) => pageOf(page, 5),
        fetchPeriods: async () => periods,
      });

      await timeline.reload();

      expect(timeline.items.value).toEqual([0, 1]);
      expect(timeline.periods.value).toEqual(periods);
    });

    it('keeps the photos when the periods fail', async () => {
      const timeline = useTimelineList<number>({
        pageSize: 2,
        fetch: async (page) => pageOf(page, 5),
        fetchPeriods: async () => Promise.reject(new Error('failed')),
      });

      await timeline.reload();

      expect(timeline.items.value).toEqual([0, 1]);
      expect(timeline.periods.value).toEqual([]);
    });
  });

  describe('ensureLoaded', () => {
    it('loads pages until the list holds the number of photos', async () => {
      const fetch = vi.fn<(page: PaginationParams) => Promise<PaginatedResponse<number>>>(
        async (page) => pageOf(page, 9),
      );
      const timeline = useTimelineList<number>({
        pageSize: 2,
        fetch,
        fetchPeriods: async () => [],
      });
      await timeline.reload();

      await timeline.ensureLoaded(5);

      expect(timeline.items.value).toHaveLength(6);
      expect(fetch).toHaveBeenCalledTimes(3);
    });

    it('stops at the end of the list', async () => {
      const timeline = useTimelineList<number>({
        pageSize: 2,
        fetch: async (page) => pageOf(page, 3),
        fetchPeriods: async () => [],
      });
      await timeline.reload();

      await timeline.ensureLoaded(10);

      expect(timeline.items.value).toEqual([0, 1, 2]);
    });
  });
});
