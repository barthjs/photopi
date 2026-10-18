import { apiFetch, UPLOAD_TIMEOUT_MS } from '@/api/client';
import type { OverlayResponse } from '@/api/types';

export const overlayRepository = {
  /** list returns every uploaded overlay. */
  list(): Promise<OverlayResponse[]> {
    return apiFetch('/overlays', {
      method: 'GET',
    });
  },

  /** create uploads a new overlay. */
  create(file: File): Promise<OverlayResponse> {
    const form = new FormData();
    form.append('file', file);
    return apiFetch('/overlays', {
      method: 'POST',
      body: form,
      timeout: UPLOAD_TIMEOUT_MS,
    });
  },

  /** delete removes the overlay with the given name. */
  delete(name: string): Promise<void> {
    return apiFetch(`/overlays/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    });
  },

  /** downloadUrl returns the address that downloads the overlay as it is stored. */
  downloadUrl(name: string): string {
    return `/api/overlays/${encodeURIComponent(name)}`;
  },
};
