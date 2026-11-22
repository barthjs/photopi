import type { ShortcutMap } from '@/lib/editor/shortcuts';

/**
 * The keyboard shortcuts of the layout editor beside those every editor offers.
 * Tab steps through the slots while no control has the focus, see useEditorWorkspace.
 */
export const LAYOUT_EDITOR_SHORTCUTS = {
  copy: ['meta', 'c'],
  paste: ['meta', 'v'],
  duplicate: ['meta', 'd'],
  nextSlot: ['tab'],
  previousSlot: ['shift', 'tab'],
  addSlot: ['n'],
  removeSlots: ['delete'],
  removeSlotsAlternative: ['backspace'],
  lock: ['l'],
} as const satisfies ShortcutMap;

export type LayoutEditorShortcut = keyof typeof LAYOUT_EDITOR_SHORTCUTS;
