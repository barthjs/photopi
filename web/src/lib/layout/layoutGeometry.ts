import type { LayoutPaperRequest, LayoutSlotRequest } from '@/api/types/layout';
import { clamp, type Bounds, type Size } from '@/lib/editor/geometry';

/** Rounds to whole millimeters, the finest precision that matters on a printed page. */
export function roundMm(value: number): number {
  return Math.round(value);
}

/**
 * Rounds the position and size of a slot to whole millimeters on the given paper.
 * Dragging moves in fractions for smooth motion, so the slot is snapped once the gesture ends.
 */
export function snapSlotToWholeMm(
  slot: LayoutSlotRequest,
  paper: LayoutPaperRequest,
  minSizeFraction: number,
): void {
  const { widthMM, heightMM } = paper;
  slot.w = clamp(Math.round(slot.w * widthMM) / widthMM, minSizeFraction, 1 - slot.x);
  slot.h = clamp(Math.round(slot.h * heightMM) / heightMM, minSizeFraction, 1 - slot.y);
  slot.x = clamp(Math.round(slot.x * widthMM) / widthMM, 0, 1 - slot.w);
  slot.y = clamp(Math.round(slot.y * heightMM) / heightMM, 0, 1 - slot.h);
}

/** Returns the size of the paper in millimeters, the content units of the layout editor. */
export function paperSize(paper: LayoutPaperRequest): Size {
  return {
    width: paper.widthMM,
    height: paper.heightMM,
  };
}

/** Returns where a slot lies on the paper, in millimeters. */
export function slotBounds(slot: LayoutSlotRequest, paper: LayoutPaperRequest): Bounds {
  return {
    minX: slot.x * paper.widthMM,
    maxX: (slot.x + slot.w) * paper.widthMM,
    minY: slot.y * paper.heightMM,
    maxY: (slot.y + slot.h) * paper.heightMM,
  };
}

/** Reads a width to height ratio written with a colon, a slash, or as a number, or returns null. */
export function parseRatio(input: string): number | null {
  const [rawWidth, rawHeight] = input.trim().split(/[:/]/);
  const width = Number(rawWidth);
  const height = rawHeight === undefined ? 1 : Number(rawHeight);
  if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) {
    return null;
  }
  return width / height;
}
