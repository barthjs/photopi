import { computed, ref, type Ref } from 'vue';

import type { PaginatedResponse, PaginationParams } from '@/api/client/list';
import { useApiError } from '@/composables/form/useApiError';

/** Describe a list that grows by a page at a time. */
export type UseInfiniteListOptions<T> = {
  pageSize: number;
  fetch: (page: PaginationParams) => Promise<PaginatedResponse<T>>;
  // Runs for the records of every page before the page counts as loaded, such as to load what belongs to them
  onPage?: (items: T[]) => Promise<void>;
  // Whether the list may load, such as once the user is known.
  // A list that may not load stays as it is.
  enabled?: () => boolean;
};

/** Loads a list page by page, each appended to the ones before, keeping only the answer to the latest request. */
export function useInfiniteList<T>(options: UseInfiniteListOptions<T>) {
  const { messageFor } = useApiError();

  // eslint-disable-next-line @typescript-eslint/no-unsafe-type-assertion -- the items hold no refs, so ref keeps their type
  const items = ref<T[]>([]) as Ref<T[]>;
  const total = ref(0);
  const isLoading = ref(false);
  const errorMessage = ref<string | null>(null);
  const hasMore = computed(() => items.value.length < total.value);

  let latest = 0;
  let failedReset = false;

  /** Fetches the next page, or the first page in place of the list on a reset. */
  async function load(reset: boolean): Promise<void> {
    if (options.enabled && !options.enabled()) return;
    // A page appended during a reload would take its offset from the list the reload replaces
    if (!reset && isLoading.value) return;

    const request = ++latest;
    isLoading.value = true;
    errorMessage.value = null;
    try {
      const page = await options.fetch({
        limit: options.pageSize,
        offset: reset ? 0 : items.value.length,
      });
      if (request !== latest) return;
      items.value = reset ? page.data : [...items.value, ...page.data];
      total.value = page.meta.totalCount;
      await options.onPage?.(page.data);
    } catch (err) {
      if (request !== latest) return;
      errorMessage.value = messageFor(err);
      failedReset = reset;
    } finally {
      if (request === latest) isLoading.value = false;
    }
  }

  /** Loads the list from its first page. */
  async function reload(): Promise<void> {
    return load(true);
  }

  /** Appends the next page to the list. */
  async function loadMore(): Promise<void> {
    return load(false);
  }

  /** Repeats the request that failed, a reload or the next page. */
  async function retry(): Promise<void> {
    return load(failedReset);
  }

  /** Takes the records that match out of the list and its total. */
  function remove(match: (item: T) => boolean): void {
    const kept = items.value.filter((item) => !match(item));
    total.value -= items.value.length - kept.length;
    items.value = kept;
  }

  return {
    items,
    total,
    isLoading,
    errorMessage,
    hasMore,
    reload,
    loadMore,
    retry,
    remove,
  };
}
