import type { LayoutPaperRequest, LayoutSlotRequest } from '@/api/types';
import { clamp, type Bounds, type Size } from '@/lib/editor/geometry';

/** roundMm rounds to whole millimeters, the finest precision that matters on a printed page. */
export function roundMm(value: number): number {
  return Math.round(value);
}

/** slugify turns a name into an ID of lowercase letters, digits, and hyphens. */
export function slugify(input: string): string {
  return input
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

/**
 * snapSlotToWholeMm rounds the position and size of a slot to whole millimeters on the given paper.
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

/** paperSize returns the size of the paper in millimeters, the content units of the layout editor. */
export function paperSize(paper: LayoutPaperRequest): Size {
  return {
    width: paper.widthMM,
    height: paper.heightMM,
  };
}

/** slotBounds returns where a slot lies on the paper, in millimeters. */
export function slotBounds(slot: LayoutSlotRequest, paper: LayoutPaperRequest): Bounds {
  return {
    minX: slot.x * paper.widthMM,
    maxX: (slot.x + slot.w) * paper.widthMM,
    minY: slot.y * paper.heightMM,
    maxY: (slot.y + slot.h) * paper.heightMM,
  };
}

/** parseRatio reads a width to height ratio such as "4:3", "4/3", or "1.33", or returns null. */
export function parseRatio(input: string): number | null {
  const [rawWidth, rawHeight] = input.trim().split(/[:/]/);
  const width = Number(rawWidth);
  const height = rawHeight === undefined ? 1 : Number(rawHeight);
  if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) {
    return null;
  }
  return width / height;
}
