import { computed, watch } from 'vue';

import { useRatioLock } from '@/composables/useRatioLock';

type NumberField = {
  get: () => number;
  set: (value: number) => void;
};

/** gcd returns the greatest common divisor of two whole numbers. */
function gcd(a: number, b: number): number {
  return b === 0 ? a : gcd(b, a % b);
}

/**
 * ratioLabel reduces a width and height to a "w:h" label, or a placeholder while either is not positive.
 */
function ratioLabel(width: number, height: number): string {
  if (!width || !height) return '–';
  const divisor = gcd(Math.round(width), Math.round(height));
  return divisor === 0 ? `${width}:${height}` : `${width / divisor}:${height / divisor}`;
}

/**
 * useAspectRatioLock keeps a width and height field at the ratio they had when it was locked,
 * so changing one recomputes the other.
 */
export function useAspectRatioLock(widthField: NumberField, heightField: NumberField) {
  const {
    locked,
    ratio,
    toggle: toggleRatio,
    lock: lockRatio,
    heightFor,
    widthFor,
  } = useRatioLock();
  let applying = false;

  /** lock locks the ratio at its current value, also while it is locked at another one. */
  function lock(): void {
    const height = heightField.get();
    if (height > 0) lockRatio(widthField.get() / height);
  }

  /** toggle locks or unlocks the ratio at its current value. */
  function toggle(): void {
    const height = heightField.get();
    toggleRatio(height > 0 ? widthField.get() / height : ratio.value);
  }

  watch(widthField.get, (value) => {
    if (!locked.value || applying) return;
    applying = true;
    heightField.set(Math.max(1, Math.round(heightFor(value))));
    applying = false;
  });

  watch(heightField.get, (value) => {
    if (!locked.value || applying) return;
    applying = true;
    widthField.set(Math.max(1, Math.round(widthFor(value))));
    applying = false;
  });

  const label = computed(() => ratioLabel(widthField.get(), heightField.get()));

  return {
    locked,
    lock,
    toggle,
    label,
  };
}
