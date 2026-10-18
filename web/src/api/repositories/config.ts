import { apiFetch } from '@/api/client';
import type {
  CloudShareSettings,
  ConfigResponse,
  ConfigSaveRequest,
  ConfigTestResponse,
  StorageSettings,
} from '@/api/types';

export const configRepository = {
  /** get returns the configuration without the values of its secrets. */
  get(): Promise<ConfigResponse> {
    return apiFetch('/config', {
      method: 'GET',
    });
  },

  /** save writes the configuration to the config file. */
  save(payload: ConfigSaveRequest): Promise<ConfigResponse> {
    return apiFetch('/config', {
      method: 'PUT',
      body: payload,
    });
  },

  /** testStorage checks that a storage configuration is reachable, without saving it. */
  testStorage(settings: StorageSettings): Promise<ConfigTestResponse> {
    return apiFetch('/config/storage/test', {
      method: 'POST',
      body: settings,
    });
  },

  /** testShare checks that a cloud share configuration is reachable, without saving it. */
  testShare(settings: CloudShareSettings): Promise<ConfigTestResponse> {
    return apiFetch('/config/share/test', {
      method: 'POST',
      body: settings,
    });
  },
};
