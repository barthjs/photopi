import type { useToast } from '@nuxt/ui/composables';
import { vi } from 'vitest';

import type { layoutRepository } from '@/api/repositories/layout';
import type { downloadJSON } from '@/lib/download';
import type { KeyValueStore } from '@/lib/keyValueStore';
import type { ConfirmOptions } from '@/composables/ui/useConfirm';

/**
 * The mocks the layout form needs, shared by the tests of the editor composables. Each test file registers them
 * with vi.mock and factories importing this module, since vi.mock only works in the test file.
 * It imports only types of the app, so the mock factories cannot form an import cycle.
 */
export const layoutFormMocks = {
  toast: {
    add: vi.fn<ReturnType<typeof useToast>['add']>(),
  },
  confirm: vi.fn<(options: ConfirmOptions) => Promise<boolean>>(async () => true),
  layouts: {
    list: vi.fn<typeof layoutRepository.list>(async () => []),
    get: vi.fn<typeof layoutRepository.get>(),
    save: vi.fn<typeof layoutRepository.save>(),
    delete: vi.fn<typeof layoutRepository.delete>(async () => {}),
  },
  download: vi.fn<typeof downloadJSON>(),
  images: {
    get: vi.fn<KeyValueStore<Blob>['get']>(async () => null),
    set: vi.fn<KeyValueStore<Blob>['set']>(async () => {}),
    delete: vi.fn<KeyValueStore<Blob>['delete']>(async () => {}),
  },
};

/** resetLayoutFormMocks restores the default behavior of every layout form mock. */
export function resetLayoutFormMocks(): void {
  layoutFormMocks.toast.add.mockReset();
  layoutFormMocks.confirm.mockReset().mockResolvedValue(true);
  layoutFormMocks.layouts.list.mockReset().mockResolvedValue([]);
  layoutFormMocks.layouts.get.mockReset();
  layoutFormMocks.layouts.save.mockReset();
  layoutFormMocks.layouts.delete.mockReset().mockResolvedValue(undefined);
  layoutFormMocks.download.mockReset();
  layoutFormMocks.images.get.mockReset().mockResolvedValue(null);
  layoutFormMocks.images.set.mockReset().mockResolvedValue(undefined);
  layoutFormMocks.images.delete.mockReset().mockResolvedValue(undefined);
}
