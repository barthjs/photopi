import { apiFetch } from '@/api/client/client';
import type { RunCreateRequest, RunResponse } from '@/api/types/run';

export const runRepository = {
  /** Starts a new run on the screen of the client. */
  async create(payload: RunCreateRequest): Promise<RunResponse> {
    return apiFetch('/camera/runs', {
      method: 'POST',
      body: payload,
    });
  },

  /** Cancels a run and discards its captures. */
  async abort(run: number): Promise<void> {
    return apiFetch(`/camera/runs/${run}`, {
      method: 'DELETE',
    });
  },
};
