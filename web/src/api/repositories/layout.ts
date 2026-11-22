import { apiFetch } from '@/api/client/client';
import type { LayoutDetailResponse, LayoutResponse, LayoutSaveRequest } from '@/api/types/layout';

export const layoutRepository = {
  /** Returns a summary of every print layout. */
  async list(): Promise<LayoutResponse[]> {
    return apiFetch('/print/layouts', {
      method: 'GET',
    });
  },

  /** Returns the layout with the given name, including its pages and slots. */
  async get(name: string): Promise<LayoutDetailResponse> {
    return apiFetch(`/print/layouts/${encodeURIComponent(name)}`, {
      method: 'GET',
    });
  },

  /** Creates or replaces the layout with the given name. */
  async save(name: string, payload: LayoutSaveRequest): Promise<LayoutDetailResponse> {
    return apiFetch(`/print/layouts/${encodeURIComponent(name)}`, {
      method: 'PUT',
      body: payload,
    });
  },

  /** Removes the layout with the given name. */
  async delete(name: string): Promise<void> {
    return apiFetch(`/print/layouts/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    });
  },
};
