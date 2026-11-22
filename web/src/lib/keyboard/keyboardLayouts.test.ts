import { describe, expect, it } from 'vitest';

import {
  type KeyboardLayout,
  keyboardLayouts,
  symbolLayout,
  textLayouts,
} from '@/lib/keyboard/keyboardLayouts';

/** Returns the number of columns every row of a layout fills. */
function rowWidths(layout: KeyboardLayout): number[] {
  return layout.map((row) => row.reduce((sum, key) => sum + key.span, 0));
}

describe('keyboardLayouts', () => {
  it.each(Object.entries(keyboardLayouts))(
    'fills the twelve columns of every row of the %s layout',
    (_, layout) => {
      expect(rowWidths(layout)).toEqual(layout.map(() => 12));
    },
  );
});

describe('textLayouts', () => {
  it.each(Object.entries(textLayouts))(
    'fills the twelve columns of every row of the %s layout',
    (_, layout) => {
      expect(rowWidths(layout)).toEqual(layout.map(() => 12));
    },
  );
});

describe('symbolLayout', () => {
  it('fills the twelve columns of every row', () => {
    expect(rowWidths(symbolLayout)).toEqual(symbolLayout.map(() => 12));
  });

  it.each(Object.entries(textLayouts))(
    'has as many rows as the %s text layout, so switching keeps the height',
    (_, layout) => {
      expect(symbolLayout).toHaveLength(layout.length);
    },
  );
});
