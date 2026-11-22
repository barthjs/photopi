import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useListQuery } from '@/composables/list/useListQuery';
import { withSetup } from '@/test/withSetup';

type Filters = {
  status?: string;
  active?: boolean;
  user?: number;
};

// The route and router are fakes, so the URL state of a test is just a reactive query object
const { route, router } = await vi.hoisted(async () => {
  const { reactive } = await import('vue');
  const fakeRoute = reactive({
    query: {},
  });
  return {
    route: fakeRoute,
    router: {
      push: vi.fn<(location: { query: Record<string, string> }) => void>(({ query }) => {
        fakeRoute.query = query;
      }),
      replace: vi.fn<(location: { query: Record<string, string> }) => void>(({ query }) => {
        fakeRoute.query = query;
      }),
    },
  };
});

vi.mock('vue-router', () => ({
  useRoute: () => route,
  useRouter: () => router,
}));

/** Creates the list query inside a component. */
function setup() {
  return withSetup(() =>
    useListQuery<Filters>({
      filterTypes: {
        status: 'string',
        active: 'boolean',
        user: 'number',
      },
      defaultSort: {
        column: 'createdAt',
        direction: 'desc',
      },
    }),
  ).result;
}

describe('useListQuery', () => {
  beforeEach(() => {
    route.query = {};
    router.push.mockClear();
  });

  describe('filters', () => {
    it('reads the filters of the URL by their types', () => {
      route.query = {
        status: 'open',
        active: 'true',
        user: '7',
      };

      const list = setup();

      expect(list.filters.value).toEqual({
        status: 'open',
        active: true,
        user: 7,
      });
    });
  });

  describe('setFilter', () => {
    it('sets the filter and returns to the first page', () => {
      route.query = {
        page: '2',
      };
      const list = setup();

      list.setFilter('status', 'open');

      expect(route.query).toEqual({
        status: 'open',
      });
    });
  });

  describe('sortParam', () => {
    it('returns the default sort as the server reads it', () => {
      const list = setup();

      expect(list.sortParam.value).toBe('-createdAt');
    });

    it('returns the sort of the URL', () => {
      route.query = {
        sort: 'takenAt',
      };

      const list = setup();

      expect(list.sortParam.value).toBe('takenAt');
    });
  });

  describe('resetFilters', () => {
    it('removes every filter', () => {
      route.query = {
        status: 'open',
        user: '7',
      };
      const list = setup();

      list.resetFilters();

      expect(route.query).toEqual({});
      expect(list.hasActiveFilters.value).toBe(false);
    });
  });
});
