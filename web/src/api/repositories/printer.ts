import { apiFetch } from '@/api/client';
import type {
  ConfigTestResponse,
  PrintCancelRequest,
  PrinterResponse,
  PrintResumeRequest,
  PrintTestRequest,
} from '@/api/types';

export const printerRepository = {
  /** list returns the printers the print system knows. */
  list(): Promise<PrinterResponse[]> {
    return apiFetch('/print/printers', {
      method: 'GET',
    });
  },

  /** test checks whether the given printer accepts jobs. */
  test(payload: PrintTestRequest): Promise<ConfigTestResponse> {
    return apiFetch('/print/test', {
      method: 'POST',
      body: payload,
    });
  },

  /** resume enables a stopped printer queue again and releases its held jobs. */
  resume(payload: PrintResumeRequest): Promise<void> {
    return apiFetch('/print/resume', {
      method: 'POST',
      body: payload,
    });
  },

  /** cancel cancels every job of a printer queue. */
  cancel(payload: PrintCancelRequest): Promise<void> {
    return apiFetch('/print/cancel', {
      method: 'POST',
      body: payload,
    });
  },
};
