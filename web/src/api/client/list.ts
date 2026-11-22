// The list format of the API: paginated responses, list queries, and bulk results

/** Describes which part of the full list a page holds. */
export type PaginatedMeta = {
  totalCount: number;
  limit: number;
  offset: number;
};

/** One page of a list. */
export type PaginatedResponse<T> = {
  data: T[];
  meta: PaginatedMeta;
};

/** Selects a page of a list. */
export type PaginationParams = {
  limit?: number;
  offset?: number;
};

/** The filters of a list query, which may hold lists of values. */
export type ListFilters = Record<string, string | number | boolean | string[] | undefined>;

/** Filter a list by the creation date of its records, both ends as 'YYYY-MM-DD'. */
export type CreatedRangeFilters = {
  createdFrom?: string;
  createdTo?: string;
};

/** The query of a list endpoint, whose sort marks a descending field with a "-" prefix. */
export type ListQuery<TFilters> = {
  filters?: TFilters;
  sort?: string;
  limit?: number;
  offset?: number;
};

/** Runs an action on several records. */
export type BulkRequest<TAction extends string> = {
  action: TAction;
  ids: string[];
};

/**
 * The result of a bulk action. A failed item does not abort the action,
 * its ID is reported so it can be shown or retried.
 */
export type BulkResponse = {
  succeeded: number;
  failed: number;
  skipped: number;
  failedIds: string[];
};

/**
 * Turns a list query into JSON:API style query parameters without unset and empty filters,
 * with a list filter as one comma-separated value.
 */
export function buildListQuery<TFilters extends ListFilters>(
  query: ListQuery<TFilters>,
): Record<string, string | number | boolean> {
  const params: Record<string, string | number | boolean> = {};

  for (const [key, value] of Object.entries(query.filters ?? {})) {
    if (value === undefined || value === '') continue;
    if (Array.isArray(value)) {
      if (value.length > 0) params[`filter[${key}]`] = value.join(',');
      continue;
    }
    params[`filter[${key}]`] = value;
  }

  if (query.sort) params.sort = query.sort;
  if (query.limit !== undefined) params['page[limit]'] = query.limit;
  if (query.offset !== undefined) params['page[offset]'] = query.offset;

  return params;
}

/** Appends params to path as a query string. Without params it returns path unchanged. */
export function withQuery(path: string, params: Record<string, string | number | boolean>): string {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    search.append(key, String(value));
  }

  const query = search.toString();
  return query ? `${path}?${query}` : path;
}
