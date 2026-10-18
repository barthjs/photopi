import { apiFetch } from '@/api/client';
import { buildListQuery, withQuery, type ListQuery, type PaginatedResponse } from '@/api/list';
import type { GalleryEventResponse, GalleryListFilters, GalleryRunResponse } from '@/api/types';

export const galleryRepository = {
  /** list returns a page of the runs, newest first. */
  list(query: ListQuery<GalleryListFilters>): Promise<PaginatedResponse<GalleryRunResponse>> {
    return apiFetch(withQuery('/gallery', buildListQuery(query)), {
      method: 'GET',
    });
  },

  /** delete removes a run with all its captures. */
  delete(run: number): Promise<void> {
    return apiFetch(`/gallery/${run}`, {
      method: 'DELETE',
    });
  },

  /** listEvents returns the events with stored runs, newest first. */
  listEvents(): Promise<GalleryEventResponse[]> {
    return apiFetch('/gallery/events', {
      method: 'GET',
    });
  },

  /** captureUrl returns the address of one capture of a run. */
  captureUrl(run: number, index: number): string {
    return `/api/gallery/${run}/captures/${index}`;
  },

  /** originalUrl returns the address of the original of one capture of a run, without the overlay. */
  originalUrl(run: number, index: number): string {
    return `/api/gallery/${run}/originals/${index}`;
  },

  /** zipUrl returns the address of the archive with every capture of a run, or with its originals. */
  zipUrl(run: number, originals = false): string {
    return `/api/gallery/${run}/zip${originals ? '?originals=true' : ''}`;
  },

  /** eventZipUrl returns the address of the archive with every capture of an event, or with their originals. */
  eventZipUrl(event: string, originals = false): string {
    return `/api/gallery/events/${encodeURIComponent(event)}/zip${originals ? '?originals=true' : ''}`;
  },
};
