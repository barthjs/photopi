import { apiFetch, UPLOAD_TIMEOUT_MS } from '@/api/client/client';
import type { CaptureResponse } from '@/api/types/capture';

export const captureRepository = {
  /**
   * Takes the next photo of a run. Only the webcam source passes a blob,
   * since the browser takes that photo itself.
   */
  async create(run: number, blob?: Blob): Promise<CaptureResponse> {
    if (!blob) {
      return apiFetch(`/camera/runs/${run}/captures`, {
        method: 'POST',
      });
    }

    const form = new FormData();
    form.append('file', blob, 'capture.jpg');
    return apiFetch(`/camera/runs/${run}/captures`, {
      method: 'POST',
      body: form,
      timeout: UPLOAD_TIMEOUT_MS,
    });
  },

  /** Returns the address of the screen version of one capture of a run, sized for the kiosk. */
  previewUrl(run: number, index: number): string {
    return `/api/camera/runs/${run}/captures/${index}/preview`;
  },
};
