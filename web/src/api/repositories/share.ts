import { apiFetch } from '@/api/client';
import type { ShareEmailRequest, ShareJobResponse } from '@/api/types';

export const shareRepository = {
  /** sendEmail starts mailing the captures of a run in the background, see getEmail for its result. */
  sendEmail(run: number, payload: ShareEmailRequest): Promise<void> {
    return apiFetch(`/share/${run}/email`, {
      method: 'POST',
      body: payload,
    });
  },

  /** getEmail returns the state of the latest email of a run. */
  getEmail(run: number): Promise<ShareJobResponse> {
    return apiFetch(`/share/${run}/email`, {
      method: 'GET',
    });
  },

  /** startCloudUpload starts uploading a run in the background, see getCloudUpload for its result. */
  startCloudUpload(run: number): Promise<void> {
    return apiFetch(`/share/${run}/cloud-upload`, {
      method: 'POST',
    });
  },

  /** getCloudUpload returns the state of the cloud upload of a run. */
  getCloudUpload(run: number): Promise<ShareJobResponse> {
    return apiFetch(`/share/${run}/cloud-upload`, {
      method: 'GET',
    });
  },

  /**
   * print starts printing a run in the background, see getPrint for its result.
   * A printer that cannot take the print right now answers with a conflict.
   */
  print(run: number): Promise<void> {
    return apiFetch(`/share/${run}/print`, {
      method: 'POST',
    });
  },

  /** getPrint returns the state of the latest print of a run. */
  getPrint(run: number): Promise<ShareJobResponse> {
    return apiFetch(`/share/${run}/print`, {
      method: 'GET',
    });
  },
};
