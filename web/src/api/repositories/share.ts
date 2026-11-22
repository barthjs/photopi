import { apiFetch } from '@/api/client/client';
import type { ShareEmailRequest, ShareJobResponse } from '@/api/types/share';

export const shareRepository = {
  /** Starts mailing the captures of a run in the background, see getEmail for its result. */
  async sendEmail(run: number, payload: ShareEmailRequest): Promise<void> {
    return apiFetch(`/share/${run}/email`, {
      method: 'POST',
      body: payload,
    });
  },

  /** Returns the state of the latest email of a run. */
  async getEmail(run: number): Promise<ShareJobResponse> {
    return apiFetch(`/share/${run}/email`, {
      method: 'GET',
    });
  },

  /** Starts uploading a run in the background, see getCloudUpload for its result. */
  async startCloudUpload(run: number): Promise<void> {
    return apiFetch(`/share/${run}/cloud-upload`, {
      method: 'POST',
    });
  },

  /** Returns the state of the cloud upload of a run. */
  async getCloudUpload(run: number): Promise<ShareJobResponse> {
    return apiFetch(`/share/${run}/cloud-upload`, {
      method: 'GET',
    });
  },

  /**
   * Starts printing a run in the background, see getPrint for its result.
   * A printer that cannot take the print right now answers with a conflict.
   */
  async print(run: number): Promise<void> {
    return apiFetch(`/share/${run}/print`, {
      method: 'POST',
    });
  },

  /** Returns the state of the latest print of a run. */
  async getPrint(run: number): Promise<ShareJobResponse> {
    return apiFetch(`/share/${run}/print`, {
      method: 'GET',
    });
  },
};
