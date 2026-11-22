import { apiFetch } from '@/api/client/client';
import type { ConfigTestResponse } from '@/api/types/config';
import type {
  PrintCancelRequest,
  PrinterResponse,
  PrintResumeRequest,
  PrintTestRequest,
} from '@/api/types/printer';

export const printerRepository = {
  /** Returns the printers the print system knows. */
  async list(): Promise<PrinterResponse[]> {
    return apiFetch('/print/printers', {
      method: 'GET',
    });
  },

  /** Checks whether the given printer accepts jobs. */
  async test(payload: PrintTestRequest): Promise<ConfigTestResponse> {
    return apiFetch('/print/test', {
      method: 'POST',
      body: payload,
    });
  },

  /** Enables a stopped printer queue again and releases its held jobs. */
  async resume(payload: PrintResumeRequest): Promise<void> {
    return apiFetch('/print/resume', {
      method: 'POST',
      body: payload,
    });
  },

  /** Cancels every job of a printer queue. */
  async cancel(payload: PrintCancelRequest): Promise<void> {
    return apiFetch('/print/cancel', {
      method: 'POST',
      body: payload,
    });
  },
};
