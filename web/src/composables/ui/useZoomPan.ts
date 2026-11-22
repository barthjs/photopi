import { computed, ref } from 'vue';

import {
  clampOffset,
  clampScale,
  DEFAULT_MAX_SCALE,
  DEFAULT_TOGGLE_SCALE,
  DEFAULT_WHEEL_FACTOR,
  distance,
  DOUBLE_TAP_DISTANCE,
  DOUBLE_TAP_MS,
  midpoint,
  TAP_SLOP,
  unzoomed,
  wheelScale,
  zoomAround,
  type Point,
  type Size,
  type ZoomPanOptions,
  type ZoomPanState,
} from '@/lib/ui/zoomPan';

/**
 * Zooms and pans content with a pinch, a double tap or click, and the wheel, with the handlers on the element
 * that keeps its size and the transform on the content, and pans with one pointer only while zoomed.
 */
export function useZoomPan(options: ZoomPanOptions = {}) {
  const maxScale = options.maxScale ?? DEFAULT_MAX_SCALE;
  const toggleScale = options.toggleScale ?? DEFAULT_TOGGLE_SCALE;
  const wheelFactor = options.wheelFactor ?? DEFAULT_WHEEL_FACTOR;

  const state = ref<ZoomPanState>(unzoomed());
  const pointers = new Map<number, Point>();
  const pointerCount = ref(0);

  let pinchStart: {
    distance: number;
    state: ZoomPanState;
  } | null = null;
  let panStart: {
    pointer: Point;
    state: ZoomPanState;
  } | null = null;
  let pressStart: Point | null = null;
  let lastTap: {
    at: number;
    point: Point;
  } | null = null;

  const isZoomed = computed(() => state.value.scale > 1);
  // A gesture of two pointers takes over before it zooms, so a swipe never starts in the middle of a pinch
  const isActive = computed(() => isZoomed.value || pointerCount.value > 1);
  const transform = computed(
    () => `translate(${state.value.x}px, ${state.value.y}px) scale(${state.value.scale})`,
  );

  /** Shows the content at the normal scale. */
  function reset(): void {
    state.value = unzoomed();
    pointers.clear();
    pointerCount.value = 0;
    pinchStart = null;
    panStart = null;
    pressStart = null;
    lastTap = null;
  }

  /** Starts a pinch with the second pointer or a pan with one pointer on zoomed content. */
  function onPointerDown(event: PointerEvent): void {
    const point = {
      x: event.clientX,
      y: event.clientY,
    };
    pointers.set(event.pointerId, point);
    pointerCount.value = pointers.size;
    pressStart = pointers.size === 1 ? point : null;

    const [a, b] = [...pointers.values()];
    if (pointers.size === 2 && a && b) {
      pinchStart = {
        distance: distance(a, b),
        state: state.value,
      };
      panStart = null;
    } else if (pointers.size === 1 && isZoomed.value) {
      panStart = {
        pointer: point,
        state: state.value,
      };
    }

    if (isActive.value) {
      event.stopPropagation();
      if (event.currentTarget instanceof Element) {
        event.currentTarget.setPointerCapture(event.pointerId);
      }
    }
  }

  /** Scales the content during a pinch and moves it during a pan. */
  function onPointerMove(event: PointerEvent): void {
    if (!pointers.has(event.pointerId)) return;
    pointers.set(event.pointerId, {
      x: event.clientX,
      y: event.clientY,
    });

    const element = event.currentTarget;
    if (!(element instanceof Element)) return;
    const [a, b] = [...pointers.values()];
    if (pinchStart && pointers.size === 2 && a && b) {
      const nextScale = clampScale(
        (pinchStart.state.scale * distance(a, b)) / Math.max(pinchStart.distance, 1),
        maxScale,
      );
      state.value = zoomAround(
        state.value,
        nextScale,
        relativeToCenter(element, midpoint(a, b)),
        sizeOf(element),
      );
      event.preventDefault();
      return;
    }

    if (panStart) {
      const size = sizeOf(element);
      state.value = {
        scale: state.value.scale,
        x: clampOffset(
          panStart.state.x + event.clientX - panStart.pointer.x,
          state.value.scale,
          size.width,
        ),
        y: clampOffset(
          panStart.state.y + event.clientY - panStart.pointer.y,
          state.value.scale,
          size.height,
        ),
      };
      event.preventDefault();
    }
  }

  /** Ends the gesture of the pointer and toggles the zoom on a double tap. */
  function onPointerUp(event: PointerEvent): void {
    if (!pointers.has(event.pointerId)) return;
    const point = {
      x: event.clientX,
      y: event.clientY,
    };
    const tapped =
      pointers.size === 1 && pressStart !== null && distance(pressStart, point) <= TAP_SLOP;
    pointers.delete(event.pointerId);
    pointerCount.value = pointers.size;
    pinchStart = null;
    pressStart = null;

    // The finger that stays after a pinch continues as a pan
    const remaining = [...pointers.values()][0];
    panStart =
      remaining && isZoomed.value
        ? {
            pointer: remaining,
            state: state.value,
          }
        : null;

    if (!tapped) return;
    const now = event.timeStamp;
    if (
      lastTap &&
      now - lastTap.at <= DOUBLE_TAP_MS &&
      distance(lastTap.point, point) <= DOUBLE_TAP_DISTANCE
    ) {
      lastTap = null;
      if (event.currentTarget instanceof Element) toggle(event.currentTarget, point);
      return;
    }
    lastTap = {
      at: now,
      point,
    };
  }

  /** Zooms into the point at the toggle scale, or back to the normal scale while zoomed. */
  function toggle(element: Element, point: Point): void {
    const nextScale = isZoomed.value ? 1 : clampScale(toggleScale, maxScale);
    state.value = zoomAround(
      state.value,
      nextScale,
      relativeToCenter(element, point),
      sizeOf(element),
    );
  }

  /** Zooms around the position of the mouse. */
  function onWheel(event: WheelEvent): void {
    event.preventDefault();
    const element = event.currentTarget;
    if (!(element instanceof Element)) return;
    const nextScale = wheelScale(state.value.scale, event.deltaY, wheelFactor, maxScale);
    state.value = zoomAround(
      state.value,
      nextScale,
      relativeToCenter(element, {
        x: event.clientX,
        y: event.clientY,
      }),
      sizeOf(element),
    );
  }

  return {
    state,
    isZoomed,
    isActive,
    transform,
    reset,
    onPointerDown,
    onPointerMove,
    onPointerUp,
    onWheel,
  };
}

/** Returns the size of an element. */
function sizeOf(element: Element): Size {
  const rect = element.getBoundingClientRect();
  return {
    width: rect.width,
    height: rect.height,
  };
}

/** Converts a position in the window into one relative to the center of an element. */
function relativeToCenter(element: Element, point: Point): Point {
  const rect = element.getBoundingClientRect();
  return {
    x: point.x - rect.left - rect.width / 2,
    y: point.y - rect.top - rect.height / 2,
  };
}
