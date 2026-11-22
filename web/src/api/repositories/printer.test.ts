import { describe, expect, it } from 'vitest';

import { printerRepository } from '@/api/repositories/printer';
import type { PrintCancelRequest, PrintResumeRequest, PrintTestRequest } from '@/api/types/printer';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('printerRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('list', () => {
    it('returns the printers', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, [
          {
            name: 'printer-1',
            status: 'idle',
          },
        ]),
      );

      const result = await printerRepository.list();

      expect(result).toEqual([
        {
          name: 'printer-1',
          status: 'idle',
        },
      ]);
      expect(lastRequest().url).toBe('/api/print/printers');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe('test', () => {
    it('posts the printer name and returns the result of the test', async () => {
      const payload: PrintTestRequest = {
        printerName: 'printer-1',
      };
      fetchMock.mockResolvedValue(
        jsonResponse(200, {
          ok: true,
        }),
      );

      const result = await printerRepository.test(payload);

      expect(result).toEqual({
        ok: true,
      });
      expect(lastRequest().url).toBe('/api/print/test');
      expect(lastRequest().init?.method).toBe('POST');
      expect(lastRequest().init?.body).toBe(JSON.stringify(payload));
    });
  });

  describe('resume', () => {
    it('posts the printer name to resume it', async () => {
      const payload: PrintResumeRequest = {
        printerName: 'printer-1',
      };
      fetchMock.mockResolvedValue(jsonResponse(204, undefined));

      await printerRepository.resume(payload);

      expect(lastRequest().url).toBe('/api/print/resume');
      expect(lastRequest().init?.method).toBe('POST');
      expect(lastRequest().init?.body).toBe(JSON.stringify(payload));
    });
  });

  describe('cancel', () => {
    it('posts the printer name to cancel its jobs', async () => {
      const payload: PrintCancelRequest = {
        printerName: 'printer-1',
      };
      fetchMock.mockResolvedValue(jsonResponse(204, undefined));

      await printerRepository.cancel(payload);

      expect(lastRequest().url).toBe('/api/print/cancel');
      expect(lastRequest().init?.method).toBe('POST');
      expect(lastRequest().init?.body).toBe(JSON.stringify(payload));
    });
  });
});
