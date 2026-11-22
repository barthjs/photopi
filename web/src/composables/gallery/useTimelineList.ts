import { ref } from 'vue';

import type { PaginatedResponse, PaginationParams } from '@/api/client/list';
import { useInfiniteList } from '@/composables/list/useInfiniteList';
import type { TimelinePeriod } from '@/lib/gallery/timeline';

/** Describe the photos of a timeline, which load page by page, and the periods of its scale. */
export type UseTimelineListOptions<T> = {
  pageSize: number;
  fetch: (page: PaginationParams) => Promise<PaginatedResponse<T>>;
  fetchPeriods: () => Promise<TimelinePeriod[]>;
  // Whether the list may load, such as once the user is known
  enabled?: () => boolean;
};

/**
 * Loads the photos of a timeline page by page together with the periods of its scale,
 * and loads further pages until a period further down is reached.
 */
export function useTimelineList<T>(options: UseTimelineListOptions<T>) {
  const list = useInfiniteList<T>({
    pageSize: options.pageSize,
    fetch: options.fetch,
    enabled: options.enabled,
  });
  const periods = ref<TimelinePeriod[]>([]);

  /** Loads the first page and the periods again. */
  async function reload(): Promise<void> {
    if (options.enabled && !options.enabled()) return;
    const [loadedPeriods] = await Promise.all([
      options.fetchPeriods().catch(() => []),
      list.reload(),
    ]);
    periods.value = loadedPeriods;
  }

  /** Loads pages until the list holds the number of photos or holds every one. */
  async function ensureLoaded(count: number): Promise<void> {
    while (list.items.value.length < count && list.hasMore.value && !list.errorMessage.value) {
      const before = list.items.value.length;
      await list.loadMore();
      // A page that brings nothing ends the loading, such as a request that another one replaced
      if (list.items.value.length === before) return;
    }
  }

  return {
    ...list,
    periods,
    reload,
    ensureLoaded,
  };
}
