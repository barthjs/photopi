import { describe, expect, it } from 'vitest';

import { systemRepository } from '@/api/repositories/system';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('systemRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('getInfo', () => {
    it('returns the system information', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, {
          hostname: 'host',
        }),
      );

      const result = await systemRepository.getInfo();

      expect(result).toEqual({
        hostname: 'host',
      });
      expect(lastRequest().url).toBe('/api/system/info');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe.each([
    {
      method: 'restartServer',
      action: () => systemRepository.restartServer(),
      url: '/api/system/server/restart',
    },
    {
      method: 'stopServer',
      action: () => systemRepository.stopServer(),
      url: '/api/system/server/stop',
    },
    {
      method: 'restartKiosk',
      action: () => systemRepository.restartKiosk(),
      url: '/api/system/kiosk/restart',
    },
    {
      method: 'stopKiosk',
      action: () => systemRepository.stopKiosk(),
      url: '/api/system/kiosk/stop',
    },
    {
      method: 'restartAll',
      action: () => systemRepository.restartAll(),
      url: '/api/system/all/restart',
    },
    {
      method: 'stopAll',
      action: () => systemRepository.stopAll(),
      url: '/api/system/all/stop',
    },
    {
      method: 'exitProcess',
      action: () => systemRepository.exitProcess(),
      url: '/api/system/process/exit',
    },
    {
      method: 'rebootHost',
      action: () => systemRepository.rebootHost(),
      url: '/api/system/host/reboot',
    },
    {
      method: 'shutdownHost',
      action: () => systemRepository.shutdownHost(),
      url: '/api/system/host/shutdown',
    },
  ])('$method', ({ action, url }) => {
    it('posts to the address of the action', async () => {
      fetchMock.mockResolvedValue(jsonResponse(202, undefined));

      await action();

      expect(lastRequest().url).toBe(url);
      expect(lastRequest().init?.method).toBe('POST');
    });
  });
});
