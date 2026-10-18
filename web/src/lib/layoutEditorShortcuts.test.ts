import { describe, expect, it } from 'vitest';

import de from '@/locales/de.json';
import en from '@/locales/en.json';
import { EDITOR_SHORTCUTS, shortcutDefinition } from '@/lib/editor/shortcuts';
import { LAYOUT_EDITOR_SHORTCUTS } from '@/lib/layoutEditorShortcuts';

describe('LAYOUT_EDITOR_SHORTCUTS', () => {
  const definitions = Object.values(LAYOUT_EDITOR_SHORTCUTS).map(shortcutDefinition);

  it('gives every action its own shortcut, also beside those of every editor', () => {
    const all = [...Object.values(EDITOR_SHORTCUTS).map(shortcutDefinition), ...definitions];

    const unique = new Set(all);

    expect(unique.size).toBe(all.length);
  });

  it.each(Object.keys(LAYOUT_EDITOR_SHORTCUTS))(
    'describes %s in the guide of every locale',
    (shortcut) => {
      expect(en.pages.layouts.guide.shortcuts).toHaveProperty(shortcut);
      expect(de.pages.layouts.guide.shortcuts).toHaveProperty(shortcut);
    },
  );

  it('leaves the digits and the arrow keys to the slots', () => {
    const taken = definitions.filter((definition) => /^\d$|arrow/.test(definition));

    expect(taken).toEqual([]);
  });
});
