// The list format of the API: paginated responses, list queries, and bulk results

/** PaginatedMeta describes which part of the full list a page holds. */
export type PaginatedMeta = {
  totalCount: number;
  limit: number;
  offset: number;
};

/** PaginatedResponse is one page of a list. */
export type PaginatedResponse<T> = {
  data: T[];
  meta: PaginatedMeta;
};

/** PaginationParams selects a page of a list. */
export type PaginationParams = {
  limit?: number;
  offset?: number;
};

/** ListQuery is the query of a list endpoint, whose sort marks a descending field with a "-" prefix. */
export type ListQuery<TFilters> = {
  filters?: TFilters;
  sort?: string;
  limit?: number;
  offset?: number;
};

/**
 * BulkResponse is the result of a bulk action. A failed item does not abort the action,
 * its ID is reported so it can be shown or retried.
 */
export type BulkResponse = {
  succeeded: number;
  failed: number;
  skipped: number;
  failedIds: string[];
};

/** buildListQuery turns a list query into the JSON:API style query parameters, leaving out unset and empty filters. */
export function buildListQuery<
  TFilters extends Record<string, string | number | boolean | undefined>,
>(query: ListQuery<TFilters>): Record<string, string | number | boolean> {
  const params: Record<string, string | number | boolean> = {};

  for (const [key, value] of Object.entries(query.filters ?? {})) {
    if (value === undefined || value === '') continue;
    params[`filter[${key}]`] = value;
  }

  if (query.sort) params.sort = query.sort;
  if (query.limit !== undefined) params['page[limit]'] = query.limit;
  if (query.offset !== undefined) params['page[offset]'] = query.offset;

  return params;
}

/** withQuery appends params to path as a query string. Without params it returns path unchanged. */
export function withQuery(path: string, params: Record<string, string | number | boolean>): string {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    search.append(key, String(value));
  }

  const query = search.toString();
  return query ? `${path}?${query}` : path;
}
