import type { useToast } from '@nuxt/ui/composables';
import { vi } from 'vitest';

import type { layoutRepository } from '@/api/repositories/layout';
import type { downloadJSON } from '@/lib/browser/download';
import type { KeyValueStore } from '@/lib/browser/keyValueStore';
import type { ConfirmOptions } from '@/composables/ui/useConfirm';

/**
 * The mocks the layout form needs, which each test file registers with vi.mock and a factory importing this module,
 * importing only types of the app, so the factories cannot form an import cycle.
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

/** Restores the default behavior of every layout form mock. */
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
