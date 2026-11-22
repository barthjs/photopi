import { describe, expect, it, vi } from 'vitest';

import { useZoomPan } from '@/composables/ui/useZoomPan';

// The element is 200 by 100 pixels with its top-left corner at the origin of the window
const element = document.createElement('div');
element.getBoundingClientRect = () =>
  DOMRect.fromRect({
    width: 200,
    height: 100,
  });
element.setPointerCapture = vi.fn<(pointerId: number) => void>();

/** Builds a pointer event on the element. */
function pointer(pointerId: number, clientX: number, clientY: number, timeStamp = 0): PointerEvent {
  const event = new PointerEvent('pointermove', {
    pointerId,
    clientX,
    clientY,
  });
  Object.defineProperty(event, 'currentTarget', {
    value: element,
  });
  Object.defineProperty(event, 'timeStamp', {
    value: timeStamp,
  });
  vi.spyOn(event, 'preventDefault');
  vi.spyOn(event, 'stopPropagation');
  return event;
}

/** Builds a wheel event on the element. */
function wheel(deltaY: number, clientX: number, clientY: number): WheelEvent {
  const event = new WheelEvent('wheel', {
    deltaY,
  });
  // happy-dom ignores the position in the init of a wheel event
  Object.defineProperties(event, {
    clientX: {
      value: clientX,
    },
    clientY: {
      value: clientY,
    },
    currentTarget: {
      value: element,
    },
  });
  vi.spyOn(event, 'preventDefault');
  return event;
}

/** Presses and releases one pointer at a position. */
function tap(
  zoomPan: ReturnType<typeof useZoomPan>,
  x: number,
  y: number,
  timeStamp: number,
): void {
  zoomPan.onPointerDown(pointer(1, x, y, timeStamp));
  zoomPan.onPointerUp(pointer(1, x, y, timeStamp));
}

describe('useZoomPan', () => {
  describe('reset', () => {
    it('shows the content at the normal scale', () => {
      const zoomPan = useZoomPan();
      zoomPan.state.value = {
        scale: 2,
        x: 30,
        y: 10,
      };

      zoomPan.reset();

      expect(zoomPan.state.value).toEqual({
        scale: 1,
        x: 0,
        y: 0,
      });
      expect(zoomPan.isZoomed.value).toBe(false);
    });
  });

  describe('onPointerDown', () => {
    it('leaves a single pointer at the normal scale to the surrounding gesture', () => {
      const zoomPan = useZoomPan();
      const event = pointer(1, 100, 50);

      zoomPan.onPointerDown(event);

      expect(zoomPan.isActive.value).toBe(false);
      expect(event.stopPropagation).not.toHaveBeenCalled();
    });

    it('takes over the gesture with a second pointer', () => {
      const zoomPan = useZoomPan();
      zoomPan.onPointerDown(pointer(1, 80, 50));
      const event = pointer(2, 120, 50);

      zoomPan.onPointerDown(event);

      expect(zoomPan.isActive.value).toBe(true);
      expect(event.stopPropagation).toHaveBeenCalled();
    });
  });

  describe('onPointerMove', () => {
    it('zooms with a pinch of two pointers', () => {
      const zoomPan = useZoomPan();
      zoomPan.onPointerDown(pointer(1, 80, 50));
      zoomPan.onPointerDown(pointer(2, 120, 50));

      zoomPan.onPointerMove(pointer(2, 160, 50));

      expect(zoomPan.state.value.scale).toBe(2);
    });

    it('pans zoomed content with one pointer', () => {
      const zoomPan = useZoomPan();
      zoomPan.state.value = {
        scale: 2,
        x: 0,
        y: 0,
      };
      zoomPan.onPointerDown(pointer(1, 100, 50));

      zoomPan.onPointerMove(pointer(1, 130, 40));

      expect(zoomPan.state.value).toEqual({
        scale: 2,
        x: 30,
        y: -10,
      });
    });

    it('does not pan at the normal scale', () => {
      const zoomPan = useZoomPan();
      zoomPan.onPointerDown(pointer(1, 100, 50));

      zoomPan.onPointerMove(pointer(1, 130, 40));

      expect(zoomPan.state.value).toEqual({
        scale: 1,
        x: 0,
        y: 0,
      });
    });
  });

  describe('onPointerUp', () => {
    it('zooms in on a double tap', () => {
      const zoomPan = useZoomPan();
      tap(zoomPan, 100, 50, 0);

      tap(zoomPan, 100, 50, 200);

      expect(zoomPan.state.value.scale).toBe(2);
    });

    it('zooms out on a double tap while zoomed', () => {
      const zoomPan = useZoomPan();
      zoomPan.state.value = {
        scale: 3,
        x: 20,
        y: 10,
      };
      tap(zoomPan, 100, 50, 0);

      tap(zoomPan, 100, 50, 200);

      expect(zoomPan.state.value).toEqual({
        scale: 1,
        x: 0,
        y: 0,
      });
    });

    it('ignores two taps too far apart in time', () => {
      const zoomPan = useZoomPan();
      tap(zoomPan, 100, 50, 0);

      tap(zoomPan, 100, 50, 1000);

      expect(zoomPan.state.value.scale).toBe(1);
    });

    it('ignores a pointer that moved before its release', () => {
      const zoomPan = useZoomPan();
      tap(zoomPan, 100, 50, 0);
      zoomPan.onPointerDown(pointer(1, 100, 50, 100));

      zoomPan.onPointerUp(pointer(1, 160, 50, 200));

      expect(zoomPan.state.value.scale).toBe(1);
    });
  });

  describe('onWheel', () => {
    it('zooms around the position of the mouse', () => {
      const zoomPan = useZoomPan({
        wheelFactor: 2,
      });
      const event = wheel(-100, 140, 70);

      zoomPan.onWheel(event);

      expect(zoomPan.state.value).toEqual({
        scale: 2,
        x: -40,
        y: -20,
      });
      expect(event.preventDefault).toHaveBeenCalled();
    });
  });
});
