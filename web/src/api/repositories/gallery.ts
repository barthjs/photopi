import { apiFetch } from '@/api/client/client';
import {
  buildListQuery,
  withQuery,
  type BulkRequest,
  type BulkResponse,
  type ListQuery,
  type PaginatedResponse,
} from '@/api/client/list';
import type {
  GalleryDayListResponse,
  GalleryEventResponse,
  GalleryListFilters,
  GalleryRunResponse,
} from '@/api/types/gallery';

export const galleryRepository = {
  /** Returns a page of the runs, newest first. */
  async list(query: ListQuery<GalleryListFilters>): Promise<PaginatedResponse<GalleryRunResponse>> {
    return apiFetch(withQuery('/gallery', buildListQuery(query)), {
      method: 'GET',
    });
  },

  /** Returns the number of runs that match the filters per day, the newest day first. */
  async listDays(filters: GalleryListFilters): Promise<GalleryDayListResponse> {
    return apiFetch(
      withQuery(
        '/gallery/days',
        buildListQuery({
          filters,
        }),
      ),
      {
        method: 'GET',
      },
    );
  },

  /** Removes a run with all its captures. */
  async delete(run: number): Promise<void> {
    return apiFetch(`/gallery/${run}`, {
      method: 'DELETE',
    });
  },

  /** Runs an action on several runs, given by their numbers. */
  async bulk(payload: BulkRequest<'delete'>): Promise<BulkResponse> {
    return apiFetch('/gallery/bulk', {
      method: 'POST',
      body: payload,
    });
  },

  /** Returns the events with stored runs, newest first. */
  async listEvents(): Promise<GalleryEventResponse[]> {
    return apiFetch('/gallery/events', {
      method: 'GET',
    });
  },

  /** Returns the address of one capture of a run. */
  captureUrl(run: number, index: number): string {
    return `/api/gallery/${run}/captures/${index}`;
  },

  /** Returns the address of the thumbnail of a run, for a tile. */
  thumbnailUrl(run: number): string {
    return `/api/gallery/${run}/thumbnail`;
  },

  /** Returns the address of the original of one capture of a run, without the overlay. */
  originalUrl(run: number, index: number): string {
    return `/api/gallery/${run}/originals/${index}`;
  },

  /** Returns the address of the archive with every capture of a run, or with its originals. */
  zipUrl(run: number, originals = false): string {
    return `/api/gallery/${run}/zip${originals ? '?originals=true' : ''}`;
  },

  /** Returns the address of the archive with every capture of the runs, or with their originals. */
  selectionZipUrl(runs: number[], originals = false): string {
    return withQuery('/api/gallery/zip', {
      'filter[run]': runs.join(','),
      ...(originals
        ? {
            originals: true,
          }
        : {}),
    });
  },

  /** Returns the address of the archive with every capture of an event, or with their originals. */
  eventZipUrl(event: string, originals = false): string {
    return `/api/gallery/events/${encodeURIComponent(event)}/zip${originals ? '?originals=true' : ''}`;
  },
};
