import { afterEach, describe, expect, it, vi } from 'vitest';

import { useOnScreenKeyboard } from '@/composables/kiosk/useOnScreenKeyboard';

/** Makes the media query of a coarse pointer match on a device that is used by touch. */
function stubPointer(touch: boolean): void {
  vi.stubGlobal(
    'matchMedia',
    vi.fn(() => ({
      matches: touch,
    })),
  );
}

describe('useOnScreenKeyboard', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  describe('onFocus', () => {
    it('opens the keyboard on a device that is used by touch', () => {
      stubPointer(true);
      const keyboard = useOnScreenKeyboard();

      keyboard.onFocus();

      expect(keyboard.open.value).toBe(true);
    });

    it('leaves the keyboard closed on a device with a real keyboard', () => {
      stubPointer(false);
      const keyboard = useOnScreenKeyboard();

      keyboard.onFocus();

      expect(keyboard.open.value).toBe(false);
    });
  });

  describe('toggle', () => {
    it('opens and closes the keyboard', () => {
      stubPointer(false);
      const keyboard = useOnScreenKeyboard();

      keyboard.toggle();
      const opened = keyboard.open.value;
      keyboard.toggle();

      expect(opened).toBe(true);
      expect(keyboard.open.value).toBe(false);
    });
  });

  describe('close', () => {
    it('closes an open keyboard', () => {
      stubPointer(true);
      const keyboard = useOnScreenKeyboard();
      keyboard.onFocus();

      keyboard.close();

      expect(keyboard.open.value).toBe(false);
    });
  });
});
