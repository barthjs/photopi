/** Bound the zoom of an element. */
export type ZoomPanOptions = {
  maxScale?: number;
  // The scale a double tap or double click toggles to
  toggleScale?: number;
  // The factor one wheel step of 100 pixels zooms by
  wheelFactor?: number;
};

/** A position in pixels. */
export type Point = {
  x: number;
  y: number;
};

/** The scale of an element and its offset from the centered position. */
export type ZoomPanState = {
  scale: number;
  x: number;
  y: number;
};

/** The width and the height of the element in pixels. */
export type Size = {
  width: number;
  height: number;
};

/** The largest scale without another maximum. */
export const DEFAULT_MAX_SCALE = 4;

/** The scale a double tap toggles to without another one. */
export const DEFAULT_TOGGLE_SCALE = 2;

/** The factor of one wheel step without another one. */
export const DEFAULT_WHEEL_FACTOR = 1.2;

/** The time within which a second tap counts as a double tap. */
export const DOUBLE_TAP_MS = 300;

/** The distance in pixels within which a second tap counts as a double tap. */
export const DOUBLE_TAP_DISTANCE = 24;

/** The distance in pixels a pointer may move between press and release and still tap. */
export const TAP_SLOP = 8;

/** Returns the state of an element at the normal scale. */
export function unzoomed(): ZoomPanState {
  return {
    scale: 1,
    x: 0,
    y: 0,
  };
}

/** Keeps a scale between 1 and the maximum. */
export function clampScale(scale: number, maxScale: number): number {
  return Math.min(Math.max(scale, 1), maxScale);
}

/** Keeps an offset within the overflow of an element scaled around its center, so no edge comes inside. */
export function clampOffset(offset: number, scale: number, size: number): number {
  const limit = (size * (scale - 1)) / 2;
  return Math.min(Math.max(offset, -limit), limit);
}

/**
 * Scales an element to the next scale, keeping the point under the given position in place.
 * The point is relative to the center of the element, and the result stays within the bounds of the element.
 */
export function zoomAround(
  state: ZoomPanState,
  nextScale: number,
  point: Point,
  size: Size,
): ZoomPanState {
  if (nextScale <= 1) return unzoomed();

  const ratio = nextScale / state.scale;
  return {
    scale: nextScale,
    x: clampOffset(point.x - ratio * (point.x - state.x), nextScale, size.width),
    y: clampOffset(point.y - ratio * (point.y - state.y), nextScale, size.height),
  };
}

/** Returns the scale after a wheel step, zooming in for a negative delta. */
export function wheelScale(
  scale: number,
  deltaY: number,
  factor: number,
  maxScale: number,
): number {
  return clampScale(scale * factor ** (-deltaY / 100), maxScale);
}

/** Returns the distance between two points. */
export function distance(a: Point, b: Point): number {
  return Math.hypot(a.x - b.x, a.y - b.y);
}

/** Returns the point halfway between two points. */
export function midpoint(a: Point, b: Point): Point {
  return {
    x: (a.x + b.x) / 2,
    y: (a.y + b.y) / 2,
  };
}
