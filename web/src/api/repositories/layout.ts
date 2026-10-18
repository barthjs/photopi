import { apiFetch } from '@/api/client';
import type { LayoutDetailResponse, LayoutResponse, LayoutSaveRequest } from '@/api/types';

export const layoutRepository = {
  /** list returns a summary of every print layout. */
  list(): Promise<LayoutResponse[]> {
    return apiFetch('/print/layouts', {
      method: 'GET',
    });
  },

  /** get returns the layout with the given ID, including its pages and slots. */
  get(id: string): Promise<LayoutDetailResponse> {
    return apiFetch(`/print/layouts/${encodeURIComponent(id)}`, {
      method: 'GET',
    });
  },

  /** save creates or replaces the layout with the given ID. */
  save(id: string, payload: LayoutSaveRequest): Promise<LayoutDetailResponse> {
    return apiFetch(`/print/layouts/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: payload,
    });
  },

  /** delete removes the layout with the given ID. */
  delete(id: string): Promise<void> {
    return apiFetch(`/print/layouts/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    });
  },
};
