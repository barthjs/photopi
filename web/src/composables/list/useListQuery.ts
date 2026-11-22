import { computed, onScopeDispose, reactive, ref, watch } from 'vue';
import { useRoute, useRouter, type LocationQuery, type LocationQueryValue } from 'vue-router';

import type { TableSorting } from '@/lib/table/columns';
import { TABLE_FILTER_ALL, withoutEmpty, type TableDateRange } from '@/lib/table/filters';
import { sameTableFilters } from '@/lib/table/quickFilters';
import { queryParam, replaceRouteQuery } from '@/lib/ui/routeQuery';

/** The type a filter value is parsed to from the URL. */
export type TableFilterType = 'string' | 'boolean' | 'number';

/** The sorted column of a list and its direction. */
export type TableQuerySort = {
  column: string;
  direction: 'asc' | 'desc';
};

/** Configures a list whose search, filters, and sort live in the URL. */
export type UseListQueryOptions<
  TFilters extends Record<string, string | number | boolean | undefined>,
> = {
  // Every filter except the search "q" with the type of its value
  filterTypes: Record<Exclude<keyof TFilters, 'q'> & string, TableFilterType>;
  defaultSort: TableQuerySort;
  // Reports whether the list hides a filter, such as that of a field a page fixed it to.
  // The URL cannot set a hidden filter, so it neither filters the rows unseen nor counts as active.
  isHiddenFilter?: (key: string) => boolean;
  // Whether the query is kept in the URL, true by default. A list in a dialog keeps it in memory,
  // so it leaves the URL of the page below alone.
  inUrl?: boolean;
  // The filters a list starts with while its query sets no search, filter, or sort, such as only the open records.
  // Resetting the filters returns to them.
  defaultFilters?: () => Partial<TFilters>;
};

/** The delay before a typed search reaches the URL. */
const SEARCH_DEBOUNCE_MS = 300;

/**
 * Keeps the search, the filters, and the sort of a list in the URL, or in memory outside it,
 * where every change adds a history entry and returns to the first page.
 */
export function useListQuery<
  TFilters extends Record<string, string | number | boolean | undefined>,
>(options: UseListQueryOptions<TFilters>) {
  // A query without a search, a filter, or a sort, as of a link to the list, gets the default filters
  const route = useRoute();
  const router = useRouter();

  const inUrl = options.inUrl ?? true;
  const memory = reactive<Record<string, string | undefined>>({});

  /** Returns a parameter of the table from the URL, or from memory for a table outside the URL. */
  function readQuery(key: string): LocationQueryValue | LocationQueryValue[] | undefined {
    return inUrl ? route.query[key] : memory[key];
  }

  // The last query the table set itself, which tells its own changes from a link that empties the query
  let ownQuery: string | undefined;

  /** Changes the query and adds a history entry. */
  function pushQuery(patch: Record<string, string | undefined>): void {
    if (!inUrl) {
      Object.assign(memory, patch);
      return;
    }
    const query = mergedQuery(route.query, patch);
    ownQuery = JSON.stringify(query);
    void router.push({
      query,
    });
  }

  /** Changes the query without a history entry. */
  function replaceQuery(patch: Record<string, string | undefined>): void {
    if (!inUrl) {
      Object.assign(memory, patch);
      return;
    }
    void replaceRouteQuery(route, router, (current) => {
      const query = mergedQuery(current, patch);
      ownQuery = JSON.stringify(query);
      return query;
    });
  }

  const search = computed(() => queryParam(readQuery('q')) ?? '');

  // The URL follows the input debounced, so a search adds one history entry, not one per keystroke
  // eslint-disable-next-line vue/no-ref-object-reactivity-loss -- the input follows the search through the watch below.
  const searchInput = ref(search.value);
  let searchTimer: ReturnType<typeof setTimeout> | undefined;

  watch(search, (value) => {
    if (value !== searchInput.value) searchInput.value = value;
  });

  watch(searchInput, (value) => {
    clearTimeout(searchTimer);
    if (value === search.value) return;
    searchTimer = setTimeout(() => {
      pushQuery({
        q: value || undefined,
        page: undefined,
      });
    }, SEARCH_DEBOUNCE_MS);
  });

  onScopeDispose(() => clearTimeout(searchTimer));

  const filters = computed<Partial<TFilters>>(() => {
    const result: Record<string, string | number | boolean> = {};
    for (const [key, type] of Object.entries<TableFilterType>(options.filterTypes)) {
      if (options.isHiddenFilter?.(key)) continue;
      const raw = queryParam(readQuery(key));
      if (raw === undefined) continue;
      if (type === 'boolean') {
        if (raw === 'true') result[key] = true;
        if (raw === 'false') result[key] = false;
      } else if (type === 'number') {
        const parsed = Number(raw);
        if (!Number.isNaN(parsed)) result[key] = parsed;
      } else {
        result[key] = raw;
      }
    }
    // eslint-disable-next-line @typescript-eslint/no-unsafe-type-assertion -- the keys and their types come from the filter types
    return result as Partial<TFilters>;
  });

  /** Sets a filter and returns to the first page. A cleared control unsets it with null. */
  function setFilter(
    key: Exclude<keyof TFilters, 'q'> & string,
    value: string | number | boolean | null | undefined,
  ): void {
    pushQuery({
      [key]: value === undefined || value === null || value === '' ? undefined : String(value),
      page: undefined,
    });
  }

  /** Sets several filters with one history entry and returns to the first page. */
  function setFilters(
    values: Partial<Record<Exclude<keyof TFilters, 'q'> & string, string | number | boolean>>,
  ): void {
    pushQuery(filterPatch(values));
  }

  /** Replaces every filter with the values of a quick filter and keeps the search and the sort. */
  function applyQuickFilter(values: Partial<TFilters>): void {
    pushQuery(filterPatch(values, true));
  }

  /** Returns a filter as a writable value for the v-model of its control. */
  function bindFilter<K extends Exclude<keyof TFilters, 'q'> & string>(key: K) {
    return computed<TFilters[K] | undefined>({
      get: () => filters.value[key],
      set: (value) => setFilter(key, value),
    });
  }

  /**
   * Returns a filter as the string value of a select, TABLE_FILTER_ALL without a value.
   * The value goes through the URL as text, so a number filter reads back as a number.
   */
  function bindSelectFilter(key: Exclude<keyof TFilters, 'q'> & string) {
    return computed<string>({
      get: () => {
        const value = filters.value[key];
        return value === undefined ? TABLE_FILTER_ALL : String(value);
      },
      set: (value) => setFilter(key, value === TABLE_FILTER_ALL ? undefined : value),
    });
  }

  /** Returns two date filters as one range, so both ends change with one history entry. */
  function bindDateRange(
    fromKey: Exclude<keyof TFilters, 'q'> & string,
    toKey: Exclude<keyof TFilters, 'q'> & string,
  ) {
    return computed<TableDateRange>({
      get: () => ({
        from: String(filters.value[fromKey] ?? ''),
        to: String(filters.value[toKey] ?? ''),
      }),
      set: (range) => {
        const patch: Partial<Record<Exclude<keyof TFilters, 'q'> & string, string>> = {};
        patch[fromKey] = range.from || undefined;
        patch[toKey] = range.to || undefined;
        setFilters(patch);
      },
    });
  }

  const defaultFilters = (): Partial<TFilters> => options.defaultFilters?.() ?? {};

  // A search or filters other than the default ones, which a reset undoes
  const hasActiveFilters = computed(
    () => search.value !== '' || !sameTableFilters(filters.value, defaultFilters()),
  );

  /** Removes the search and returns every filter to its default. */
  function resetFilters(): void {
    searchInput.value = '';
    pushQuery({
      ...filterPatch(defaultFilters(), true),
      q: undefined,
    });
  }

  /**
   * Returns the change of the query that sets the filters and returns to the first page.
   * With clear, it also unsets every other filter the table shows.
   */
  function filterPatch(
    values: Partial<Record<string, string | number | boolean>>,
    clear = false,
  ): Record<string, string | undefined> {
    const patch: Record<string, string | undefined> = {
      page: undefined,
    };
    if (clear) {
      for (const key of Object.keys(options.filterTypes)) {
        if (!options.isHiddenFilter?.(key)) patch[key] = undefined;
      }
    }
    for (const [key, value] of Object.entries<string | number | boolean | undefined>(values)) {
      patch[key] = value === undefined || value === '' ? undefined : String(value);
    }
    return patch;
  }

  /** Reports whether the query sets no search, no filter, and no sort, as a link to the list does. */
  function isUntouched(): boolean {
    if (queryParam(readQuery('q')) !== undefined || queryParam(readQuery('sort')) !== undefined) {
      return false;
    }
    return Object.keys(options.filterTypes).every(
      (key) => (options.isHiddenFilter?.(key) ?? false) || queryParam(readQuery(key)) === undefined,
    );
  }

  /** Sets the default filters on an untouched query and reports whether it changed the query. */
  function applyDefaultFilters(): boolean {
    const defaults = withoutEmpty(defaultFilters());
    if (Object.keys(defaults).length === 0 || !isUntouched()) return false;
    replaceQuery(filterPatch(defaults));
    return true;
  }

  const sort = computed<TableQuerySort>(() => {
    const raw = queryParam(readQuery('sort'));
    if (!raw) return options.defaultSort;
    if (raw.startsWith('-')) {
      return {
        column: raw.slice(1),
        direction: 'desc',
      };
    }
    return {
      column: raw,
      direction: 'asc',
    };
  });

  /** Returns the sort as the server reads it, with "-" for descending order. */
  function serializeSort(value: TableQuerySort): string {
    return (value.direction === 'desc' ? '-' : '') + value.column;
  }

  // The sort as the server reads it
  const sortParam = computed(() => serializeSort(sort.value));

  // Maps the URL sort to the sorting model of UTable
  const sorting = computed<TableSorting>({
    get: () => [
      {
        id: sort.value.column,
        desc: sort.value.direction === 'desc',
      },
    ],
    set: (value) => {
      const next = value[0];
      const nextSort: TableQuerySort = next
        ? {
            column: next.id,
            direction: next.desc ? 'desc' : 'asc',
          }
        : options.defaultSort;
      const serialized = serializeSort(nextSort);
      pushQuery({
        sort: serialized === serializeSort(options.defaultSort) ? undefined : serialized,
        page: undefined,
      });
    },
  });

  // A link that empties the query, such as the one in the navigation, starts with the default filters again
  if (inUrl) {
    watch(
      () => JSON.stringify(mergedQuery(route.query, {})),
      (query) => {
        if (query !== ownQuery) applyDefaultFilters();
      },
    );
  }

  return {
    searchInput,
    search,
    filters,
    setFilter,
    setFilters,
    applyQuickFilter,
    bindFilter,
    bindSelectFilter,
    bindDateRange,
    resetFilters,
    hasActiveFilters,
    sort,
    sortParam,
    sorting,
    readQuery,
    pushQuery,
    replaceQuery,
    applyDefaultFilters,
  };
}

/** Applies a patch to a query. Keys set to undefined are removed. */
function mergedQuery(
  query: LocationQuery,
  patch: Record<string, string | undefined>,
): Record<string, string> {
  const next: Record<string, string> = {};
  for (const key of Object.keys(query)) {
    if (key in patch) continue;
    const value = queryParam(query[key]);
    if (value !== undefined) next[key] = value;
  }
  for (const [key, value] of Object.entries(patch)) {
    if (value !== undefined) next[key] = value;
  }
  return next;
}
