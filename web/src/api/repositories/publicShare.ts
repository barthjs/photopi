import { apiFetch } from '@/api/client';
import type { PublicShareResponse } from '@/api/types';

export const publicShareRepository = {
  /** get returns the signed file URLs of the shared run behind a token. */
  get(token: string): Promise<PublicShareResponse> {
    return apiFetch(`/share/public/${encodeURIComponent(token)}`, {
      method: 'GET',
    });
  },

  /** zipUrl returns the address of the archive of a shared run. */
  zipUrl(token: string): string {
    return `/api/share/public/${encodeURIComponent(token)}/zip`;
  },
};
