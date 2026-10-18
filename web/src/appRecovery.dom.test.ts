import { beforeEach, describe, expect, it, vi } from 'vitest';

import { recordReload } from '@/appRecovery';

vi.mock('@/router', () => ({
  router: {},
}));

vi.mock('@/api/repositories/meta', () => ({
  metaRepository: {},
}));

describe('recordReload', () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it('reloads quickly while reloads are rare', () => {
    const first = recordReload(1_000);

    const second = recordReload(2_000);

    expect(first).toBe(2_000);
    expect(second).toBe(2_000);
  });

  it('backs off once the reloads within a minute reach the threshold', () => {
    recordReload(1_000);
    recordReload(2_000);

    const delay = recordReload(3_000);

    expect(delay).toBe(30_000);
  });

  it('forgets reloads older than the window', () => {
    recordReload(1_000);
    recordReload(2_000);

    const delay = recordReload(70_000);

    expect(delay).toBe(2_000);
  });

  it('reloads quickly without a readable history', () => {
    sessionStorage.setItem('photopi:reloadHistory', 'not json');

    const delay = recordReload(1_000);

    expect(delay).toBe(2_000);
  });
});
