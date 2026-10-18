import { vi } from 'vitest';

import { withSetup } from '@/test/withSetup';
import type { FormErrorTarget } from '@/composables/form/useFormValidation';
import type { LayoutForm } from '@/composables/useLayoutForm';
import { provideLayoutForm } from '@/composables/useLayoutForm';

export { layoutFormMocks, resetLayoutFormMocks } from '@/test/layoutFormMocks';

/**
 * mountLayoutForm creates a layout form inside a mounted component and runs extra composables
 * in the same component.
 */
export function mountLayoutForm<T>(setup: (layoutForm: LayoutForm) => T): {
  layoutForm: LayoutForm;
  result: T;
  unmount: () => void;
} {
  const form = {
    setErrors: vi.fn<FormErrorTarget['setErrors']>(),
    clear: vi.fn<() => void>(),
  };
  let layoutForm: LayoutForm | undefined;

  const { result, unmount } = withSetup(() => {
    layoutForm = provideLayoutForm(form);
    return setup(layoutForm);
  });

  return {
    layoutForm: layoutForm as LayoutForm,
    result,
    unmount,
  };
}
