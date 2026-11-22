import { describe, expect, it } from 'vitest';

import type { LayoutResponse } from '@/api/types/layout';
import { layoutMenuItems } from '@/lib/layout/layoutItems';

describe('layoutMenuItems', () => {
  it('keeps a description and leaves out a missing one', () => {
    const layout: LayoutResponse = {
      name: 'Strip',
      description: 'Three photos',
      paperWidthMM: 50,
      paperHeightMM: 152,
      slots: 3,
      pages: 1,
      inUse: false,
    };

    const items = layoutMenuItems([
      layout,
      {
        ...layout,
        name: 'Single',
        description: null,
      },
    ]);

    expect(items).toEqual([
      layout,
      {
        name: 'Single',
        paperWidthMM: 50,
        paperHeightMM: 152,
        slots: 3,
        pages: 1,
        inUse: false,
      },
    ]);
  });
});
