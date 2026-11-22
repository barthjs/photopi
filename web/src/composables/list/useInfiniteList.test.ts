import { describe, expect, it, vi } from 'vitest';
import { faker } from '@faker-js/faker';

import type { PaginatedResponse, PaginationParams } from '@/api/client/list';
import { useInfiniteList } from '@/composables/list/useInfiniteList';

vi.mock('@/composables/form/useApiError', () => ({
  useApiError: () => ({
    messageFor: (err: unknown) => (err instanceof Error ? err.message : String(err)),
  }),
}));

/** A record of the list under test. */
type Row = {
  id: string;
};

/** Returns records with fresh IDs. */
function rows(count: number): Row[] {
  return Array.from(
    {
      length: count,
    },
    () => ({
      id: faker.string.uuid(),
    }),
  );
}

/** Returns a page of a list with the records and the total. */
function page(data: Row[], totalCount: number): PaginatedResponse<Row> {
  return {
    data,
    meta: {
      totalCount,
      limit: data.length,
      offset: 0,
    },
  };
}

describe('useInfiniteList', () => {
  describe('reload', () => {
    it('loads the first page in place of the list', async () => {
      const first = rows(2);
      const fetch = vi.fn<(page: PaginationParams) => Promise<PaginatedResponse<Row>>>(async () =>
        page(first, 5),
      );
      const list = useInfiniteList({
        pageSize: 2,
        fetch,
      });

      await list.reload();

      expect(list.items.value).toEqual(first);
      expect(list.total.value).toBe(5);
      expect(list.hasMore.value).toBe(true);
      expect(list.isLoading.value).toBe(false);
      expect(fetch).toHaveBeenCalledWith({
        limit: 2,
        offset: 0,
      });
    });

    it('runs the hook for the records of the page', async () => {
      const first = rows(2);
      const onPage = vi.fn<(records: Row[]) => Promise<void>>(async () => {});
      const list = useInfiniteList({
        pageSize: 2,
        fetch: async () => page(first, 2),
        onPage,
      });

      await list.reload();

      expect(onPage).toHaveBeenCalledWith(first);
    });

    it('keeps only the answer to the latest request', async () => {
      const stale = rows(1);
      const current = rows(1);
      let resolveStale: (value: PaginatedResponse<Row>) => void = () => {};
      const fetch = vi
        .fn<(page: PaginationParams) => Promise<PaginatedResponse<Row>>>()
        .mockReturnValueOnce(
          new Promise((resolve) => {
            resolveStale = resolve;
          }),
        )
        .mockResolvedValueOnce(page(current, 1));
      const list = useInfiniteList({
        pageSize: 1,
        fetch,
      });
      const first = list.reload();
      await list.reload();

      resolveStale(page(stale, 1));
      await first;

      expect(list.items.value).toEqual(current);
    });

    it('reports a failed request', async () => {
      const message = faker.lorem.sentence();
      const list = useInfiniteList<Row>({
        pageSize: 2,
        fetch: async () => Promise.reject(new Error(message)),
      });

      await list.reload();

      expect(list.errorMessage.value).toBe(message);
      expect(list.isLoading.value).toBe(false);
    });

    it('stays as it is while the list may not load', async () => {
      const fetch = vi.fn<(page: PaginationParams) => Promise<PaginatedResponse<Row>>>(async () =>
        page(rows(1), 1),
      );
      const list = useInfiniteList({
        pageSize: 1,
        fetch,
        enabled: () => false,
      });

      await list.reload();

      expect(fetch).not.toHaveBeenCalled();
      expect(list.items.value).toEqual([]);
    });
  });

  describe('loadMore', () => {
    it('appends the next page to the list', async () => {
      const first = rows(2);
      const second = rows(1);
      const fetch = vi
        .fn<(page: PaginationParams) => Promise<PaginatedResponse<Row>>>()
        .mockResolvedValueOnce(page(first, 3))
        .mockResolvedValueOnce(page(second, 3));
      const list = useInfiniteList({
        pageSize: 2,
        fetch,
      });
      await list.reload();

      await list.loadMore();

      expect(list.items.value).toEqual([...first, ...second]);
      expect(list.hasMore.value).toBe(false);
      expect(fetch).toHaveBeenLastCalledWith({
        limit: 2,
        offset: 2,
      });
    });

    it('waits for a running reload', async () => {
      const current = rows(1);
      let resolveReload: (value: PaginatedResponse<Row>) => void = () => {};
      const fetch = vi
        .fn<(page: PaginationParams) => Promise<PaginatedResponse<Row>>>()
        .mockReturnValueOnce(
          new Promise((resolve) => {
            resolveReload = resolve;
          }),
        );
      const list = useInfiniteList({
        pageSize: 1,
        fetch,
      });
      const reload = list.reload();

      await list.loadMore();
      resolveReload(page(current, 2));
      await reload;

      expect(fetch).toHaveBeenCalledTimes(1);
      expect(list.items.value).toEqual(current);
    });
  });

  describe('retry', () => {
    it('repeats a failed reload', async () => {
      const first = rows(1);
      const fetch = vi
        .fn<(page: PaginationParams) => Promise<PaginatedResponse<Row>>>()
        .mockRejectedValueOnce(new Error(faker.lorem.sentence()))
        .mockResolvedValueOnce(page(first, 1));
      const list = useInfiniteList({
        pageSize: 1,
        fetch,
      });
      await list.reload();

      await list.retry();

      expect(list.items.value).toEqual(first);
      expect(list.errorMessage.value).toBeNull();
      expect(fetch).toHaveBeenLastCalledWith({
        limit: 1,
        offset: 0,
      });
    });

    it('repeats a failed next page', async () => {
      const first = rows(1);
      const second = rows(1);
      const fetch = vi
        .fn<(page: PaginationParams) => Promise<PaginatedResponse<Row>>>()
        .mockResolvedValueOnce(page(first, 2))
        .mockRejectedValueOnce(new Error(faker.lorem.sentence()))
        .mockResolvedValueOnce(page(second, 2));
      const list = useInfiniteList({
        pageSize: 1,
        fetch,
      });
      await list.reload();
      await list.loadMore();

      await list.retry();

      expect(list.items.value).toEqual([...first, ...second]);
      expect(fetch).toHaveBeenLastCalledWith({
        limit: 1,
        offset: 1,
      });
    });
  });

  describe('remove', () => {
    it('takes the matching records out of the list and its total', async () => {
      const first = rows(3);
      const list = useInfiniteList({
        pageSize: 3,
        fetch: async () => page(first, 10),
      });
      await list.reload();

      list.remove((row) => row.id === first[1]?.id);

      expect(list.items.value).toEqual([first[0], first[2]]);
      expect(list.total.value).toBe(9);
    });
  });
});
