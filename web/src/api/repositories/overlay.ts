import { apiFetch, UPLOAD_TIMEOUT_MS } from '@/api/client/client';
import type { OverlayResponse } from '@/api/types/overlay';

export const overlayRepository = {
  /** Returns every uploaded overlay. */
  async list(): Promise<OverlayResponse[]> {
    return apiFetch('/overlays', {
      method: 'GET',
    });
  },

  /** Uploads a new overlay. */
  async create(file: File): Promise<OverlayResponse> {
    const form = new FormData();
    form.append('file', file);
    return apiFetch('/overlays', {
      method: 'POST',
      body: form,
      timeout: UPLOAD_TIMEOUT_MS,
    });
  },

  /** Removes the overlay with the given name. */
  async delete(name: string): Promise<void> {
    return apiFetch(`/overlays/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    });
  },

  /** Returns the address that downloads the overlay as it is stored. */
  downloadUrl(name: string): string {
    return `/api/overlays/${encodeURIComponent(name)}`;
  },
};
