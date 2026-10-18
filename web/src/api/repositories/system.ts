import { apiFetch } from '@/api/client';
import type { SystemInfoResponse } from '@/api/types';

export const systemRepository = {
  /** getInfo returns the state of the host and which system actions are enabled. */
  getInfo(): Promise<SystemInfoResponse> {
    return apiFetch('/system/info', {
      method: 'GET',
    });
  },

  /** restartServer restarts the server. */
  restartServer(): Promise<void> {
    return apiFetch('/system/server/restart', {
      method: 'POST',
    });
  },

  /** stopServer stops the server. */
  stopServer(): Promise<void> {
    return apiFetch('/system/server/stop', {
      method: 'POST',
    });
  },

  /** restartKiosk restarts the kiosk browser. */
  restartKiosk(): Promise<void> {
    return apiFetch('/system/kiosk/restart', {
      method: 'POST',
    });
  },

  /** stopKiosk stops the kiosk browser. */
  stopKiosk(): Promise<void> {
    return apiFetch('/system/kiosk/stop', {
      method: 'POST',
    });
  },

  /** restartAll restarts the server and the kiosk browser. */
  restartAll(): Promise<void> {
    return apiFetch('/system/all/restart', {
      method: 'POST',
    });
  },

  /** stopAll stops the server and the kiosk browser. */
  stopAll(): Promise<void> {
    return apiFetch('/system/all/stop', {
      method: 'POST',
    });
  },

  /** exitProcess ends the server process. */
  exitProcess(): Promise<void> {
    return apiFetch('/system/process/exit', {
      method: 'POST',
    });
  },

  /** rebootHost reboots the host. */
  rebootHost(): Promise<void> {
    return apiFetch('/system/host/reboot', {
      method: 'POST',
    });
  },

  /** shutdownHost shuts the host down. */
  shutdownHost(): Promise<void> {
    return apiFetch('/system/host/shutdown', {
      method: 'POST',
    });
  },
};
