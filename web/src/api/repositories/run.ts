import { apiFetch } from '@/api/client';
import type { RunResponse } from '@/api/types';

export const runRepository = {
  /** create starts a new run. */
  create(): Promise<RunResponse> {
    return apiFetch('/camera/runs', {
      method: 'POST',
    });
  },

  /** abort cancels a run and discards its captures. */
  abort(run: number): Promise<void> {
    return apiFetch(`/camera/runs/${run}`, {
      method: 'DELETE',
    });
  },
};
