import { apiFetch } from '@/api/client/client';
import type { PublicShareResponse } from '@/api/types/publicShare';

export const publicShareRepository = {
  /** Returns the signed file URLs of the shared run behind a token. */
  async get(token: string): Promise<PublicShareResponse> {
    return apiFetch(`/share/public/${encodeURIComponent(token)}`, {
      method: 'GET',
    });
  },

  /** Returns the address of the archive of a shared run. */
  zipUrl(token: string): string {
    return `/api/share/public/${encodeURIComponent(token)}/zip`;
  },
};
