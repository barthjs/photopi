import { describe, expect, it } from 'vitest';

import { shareRepository } from '@/api/repositories/share';
import type { ShareEmailRequest } from '@/api/types';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('shareRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('sendEmail', () => {
    it('posts the address to send the run by mail', async () => {
      const payload: ShareEmailRequest = {
        email: 'guest@example.com',
      };
      fetchMock.mockResolvedValue(jsonResponse(202, undefined));

      await shareRepository.sendEmail(4, payload);

      expect(lastRequest().url).toBe('/api/share/4/email');
      expect(lastRequest().init?.method).toBe('POST');
      expect(lastRequest().init?.body).toBe(JSON.stringify(payload));
    });
  });

  describe('getEmail', () => {
    it('returns the state of the email', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, {
          status: 'failed',
          reason: 'unreachable',
        }),
      );

      const result = await shareRepository.getEmail(4);

      expect(result).toEqual({
        status: 'failed',
        reason: 'unreachable',
      });
      expect(lastRequest().url).toBe('/api/share/4/email');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe('startCloudUpload', () => {
    it('posts to start the cloud upload', async () => {
      fetchMock.mockResolvedValue(jsonResponse(202, undefined));

      await shareRepository.startCloudUpload(4);

      expect(lastRequest().url).toBe('/api/share/4/cloud-upload');
      expect(lastRequest().init?.method).toBe('POST');
    });
  });

  describe('getCloudUpload', () => {
    it('returns the state of the cloud upload', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, {
          status: 'done',
          url: 'https://example.com/s',
        }),
      );

      const result = await shareRepository.getCloudUpload(4);

      expect(result).toEqual({
        status: 'done',
        url: 'https://example.com/s',
      });
      expect(lastRequest().url).toBe('/api/share/4/cloud-upload');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe('print', () => {
    it('posts to print the run', async () => {
      fetchMock.mockResolvedValue(jsonResponse(202, undefined));

      await shareRepository.print(4);

      expect(lastRequest().url).toBe('/api/share/4/print');
      expect(lastRequest().init?.method).toBe('POST');
    });
  });

  describe('getPrint', () => {
    it('returns the state of the print', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, {
          status: 'failed',
          reason: 'queue_unavailable',
        }),
      );

      const result = await shareRepository.getPrint(4);

      expect(result).toEqual({
        status: 'failed',
        reason: 'queue_unavailable',
      });
      expect(lastRequest().url).toBe('/api/share/4/print');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });
});
