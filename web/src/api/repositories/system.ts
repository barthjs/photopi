import { apiFetch } from '@/api/client/client';
import type { SystemInfoResponse } from '@/api/types/system';

export const systemRepository = {
  /** Returns the state of the host and which system actions are enabled. */
  async getInfo(): Promise<SystemInfoResponse> {
    return apiFetch('/system/info', {
      method: 'GET',
    });
  },

  /** Restarts the server. */
  async restartServer(): Promise<void> {
    return apiFetch('/system/server/restart', {
      method: 'POST',
    });
  },

  /** Stops the server. */
  async stopServer(): Promise<void> {
    return apiFetch('/system/server/stop', {
      method: 'POST',
    });
  },

  /** Restarts the kiosk browser. */
  async restartKiosk(): Promise<void> {
    return apiFetch('/system/kiosk/restart', {
      method: 'POST',
    });
  },

  /** Stops the kiosk browser. */
  async stopKiosk(): Promise<void> {
    return apiFetch('/system/kiosk/stop', {
      method: 'POST',
    });
  },

  /** Restarts the server and the kiosk browser. */
  async restartAll(): Promise<void> {
    return apiFetch('/system/all/restart', {
      method: 'POST',
    });
  },

  /** Stops the server and the kiosk browser. */
  async stopAll(): Promise<void> {
    return apiFetch('/system/all/stop', {
      method: 'POST',
    });
  },

  /** Ends the server process. */
  async exitProcess(): Promise<void> {
    return apiFetch('/system/process/exit', {
      method: 'POST',
    });
  },

  /** Reboots the host. */
  async rebootHost(): Promise<void> {
    return apiFetch('/system/host/reboot', {
      method: 'POST',
    });
  },

  /** Shuts the host down. */
  async shutdownHost(): Promise<void> {
    return apiFetch('/system/host/shutdown', {
      method: 'POST',
    });
  },
};
