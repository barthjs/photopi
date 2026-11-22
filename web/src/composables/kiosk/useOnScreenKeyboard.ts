import { ref, type Ref } from 'vue';

/** The state and the actions of the on-screen keyboard of a field. */
export type OnScreenKeyboard = {
  open: Ref<boolean>;
  touchDevice: boolean;
  onFocus: () => void;
  toggle: () => void;
  close: () => void;
};

/**
 * Controls the on-screen keyboard of a field in the kiosk.
 * The keyboard of the system stays hidden behind the kiosk browser in full screen,
 * so the keyboard of the application opens with the focus on a device that is used by touch
 * and can be toggled on any device.
 */
export function useOnScreenKeyboard(): OnScreenKeyboard {
  const open = ref(false);
  const touchDevice = window.matchMedia('(pointer: coarse)').matches;

  /** Opens the keyboard on a device that is used by touch. */
  function onFocus(): void {
    if (touchDevice) open.value = true;
  }

  /** Opens or closes the keyboard. */
  function toggle(): void {
    open.value = !open.value;
  }

  /** Hides the keyboard. */
  function close(): void {
    open.value = false;
  }

  return {
    open,
    touchDevice,
    onFocus,
    toggle,
    close,
  };
}
