import { ref, toRaw, toValue, type MaybeRefOrGetter } from 'vue';

import type { LayoutSlotRequest } from '@/api/types';
import { isAdditiveSelect } from '@/composables/editor/useEditorSelection';
import { MIN_SIZE_FRACTION, type LayoutForm } from '@/composables/useLayoutForm';
import { usePointerDrag } from '@/composables/usePointerDrag';
import {
  clamp,
  clampOffset,
  constrainToAxis,
  sizeBounds,
  translateBounds,
  unionBounds,
  type Bounds,
  type Point,
} from '@/lib/editor/geometry';
import {
  SNAP_DISTANCE,
  axisFeedback,
  boundsTargets,
  mergeTargets,
  noSnap,
  snapBounds,
  snapValue,
  type AxisTargets,
  type SnapFeedback,
} from '@/lib/editor/snap';
import { measureGaps, spacingTargets } from '@/lib/editor/spacing';
import { paperSize, slotBounds, snapSlotToWholeMm } from '@/lib/layoutGeometry';

/** SlotCorner is the corner of a slot a resize handle sits on. */
export type SlotCorner = 'nw' | 'ne' | 'sw' | 'se';

type SnapTargets = {
  axes: AxisTargets;
  // The bounds of the slots that stay, which the distances are measured to.
  others: Bounds[];
};

type DragState =
  | {
      mode: 'move';
      initialSlots: Map<number, LayoutSlotRequest>;
      start: Point;
      targets: SnapTargets;
    }
  | {
      mode: 'resize';
      index: number;
      corner: SlotCorner;
      // The slot at the start of the gesture in millimeters.
      initial: Bounds;
      start: Point;
      targets: SnapTargets;
    };

export type DragMode = DragState['mode'];

export type SlotDragOptions = {
  snapping: MaybeRefOrGetter<boolean>;
  // Converts a position in the browser window into millimeters on the paper.
  toPaper: (clientX: number, clientY: number) => Point;
  // The screen pixels per millimeter, so the snap reach stays the same on screen.
  scale: MaybeRefOrGetter<number>;
  // The guides the slots snap to, in millimeters.
  guides: () => AxisTargets;
};

/**
 * useSlotDrag moves slots and resizes a slot from its corners with the pointer, in millimeters on the paper.
 * A gesture records one history entry when it ends or the browser cancels it, with the moved slots
 * on whole millimeters. With snapping on, edges and centers align with those of the other slots, of the paper,
 * and with the guides, and moved slots keep even gaps to their neighbors. Holding Ctrl or Cmd suspends snapping.
 * Shift keeps a move on one axis and a resize at the ratio of the slot. Alt duplicates the slots a move starts on
 * and resizes from the center.
 */
export function useSlotDrag(options: SlotDragOptions, form: LayoutForm) {
  const { state, editor, activeSlots } = form;
  const mode = ref<DragMode | null>(null);
  // What the dragged slots or edges aligned with and their distances, shown on the canvas
  const feedback = ref<SnapFeedback>(noSnap());
  let dragState: DragState | null = null;

  const pointerDrag = usePointerDrag({
    onMove: (_dx, _dy, event) => moveTo(event),
    onEnd: () => finish(),
  });

  /**
   * targetsExcluding collects the lines of the paper, of the guides, and of every slot except the dragged ones
   * and the hidden ones.
   */
  function targetsExcluding(indices: ReadonlySet<number>): SnapTargets {
    const others = activeSlots.value
      .filter((_, index) => !indices.has(index) && !editor.visibility.isHidden(index))
      .map((slot) => slotBounds(slot, state.paper));
    return {
      axes: mergeTargets(boundsTargets(others, paperSize(state.paper)), options.guides()),
      others,
    };
  }

  /** snapDistance returns the snap reach in millimeters, or none while snapping is off or Ctrl or Cmd is held. */
  function snapDistance(event: PointerEvent): number {
    if (!toValue(options.snapping) || event.ctrlKey || event.metaKey) return 0;
    const scale = toValue(options.scale);
    return scale > 0 ? SNAP_DISTANCE / scale : SNAP_DISTANCE;
  }

  /** moveTo applies the distance the pointer traveled since the start of the gesture. */
  function moveTo(event: PointerEvent): void {
    if (!dragState) return;

    const position = options.toPaper(event.clientX, event.clientY);
    const offset = {
      x: position.x - dragState.start.x,
      y: position.y - dragState.start.y,
    };
    if (dragState.mode === 'move') {
      moveSlots(dragState, event.shiftKey ? constrainToAxis(offset) : offset, event);
      return;
    }
    resizeSlot(dragState, offset, event);
  }

  /**
   * moveSlots moves the dragged slots from their start by the pointer distance. The edges and the center of the slots
   * as a whole align with the lines in reach, an axis that aligns with nothing keeps even gaps to the neighbors,
   * and the group stops together at the paper border. A move kept on one axis only snaps along it.
   */
  function moveSlots(
    drag: Extract<DragState, { mode: 'move' }>,
    pointerOffset: Point,
    event: PointerEvent,
  ): void {
    const initialBounds = unionBounds(
      [...drag.initialSlots.values()].map((slot) => slotBounds(slot, state.paper)),
    );
    if (!initialBounds) return;

    const reach = snapDistance(event);
    const moved = translateBounds(initialBounds, pointerOffset);
    const aligned = snapBounds(moved, drag.targets.axes, reach);
    const spacing = spacingTargets(moved, drag.targets.others);
    const evenX = snapValue([moved.minX], spacing.x, reach);
    const evenY = snapValue([moved.minY], spacing.y, reach);
    const alignedX = aligned.feedback.lines.some((line) => line.axis === 'x');
    const alignedY = aligned.feedback.lines.some((line) => line.axis === 'y');

    // A move kept on one axis does not leave it for a snap
    const keepX = event.shiftKey && pointerOffset.x === 0;
    const keepY = event.shiftKey && pointerOffset.y === 0;
    const snapped = {
      x: keepX ? 0 : pointerOffset.x + (alignedX ? aligned.offset.x : evenX.delta),
      y: keepY ? 0 : pointerOffset.y + (alignedY ? aligned.offset.y : evenY.delta),
    };
    const paper = paperSize(state.paper);
    const offset = clampOffset(initialBounds, sizeBounds(paper), snapped);
    for (const [index, initial] of drag.initialSlots) {
      const slot = activeSlots.value[index];
      if (!slot) continue;
      slot.x = initial.x + offset.x / state.paper.widthMM;
      slot.y = initial.y + offset.y / state.paper.heightMM;
    }
    feedback.value = {
      ...aligned.feedback,
      gaps: measureGaps(translateBounds(initialBounds, offset), drag.targets.others, paper),
    };
  }

  /**
   * resizeSlot moves the edges at the dragged corner by the pointer distance, while the opposite edges stay,
   * or with Alt mirror them around the center. The moving edges align with the lines in reach. With Shift or
   * the ratio of the slot locked, the other side follows the side the pointer changed more, and staying
   * on the paper wins over the exact ratio.
   */
  function resizeSlot(
    drag: Extract<DragState, { mode: 'resize' }>,
    offset: Point,
    event: PointerEvent,
  ): void {
    const slot = activeSlots.value[drag.index];
    if (!slot) return;

    const { widthMM, heightMM } = state.paper;
    const start = drag.initial;
    const west = drag.corner === 'nw' || drag.corner === 'sw';
    const north = drag.corner === 'nw' || drag.corner === 'ne';
    const reach = snapDistance(event);

    const pointerX = (west ? start.minX : start.maxX) + offset.x;
    const pointerY = (north ? start.minY : start.maxY) + offset.y;
    const snapX = snapValue([pointerX], drag.targets.axes.x, reach);
    const snapY = snapValue([pointerY], drag.targets.axes.y, reach);

    let [minX, maxX] = resizeAxis(
      [start.minX, start.maxX],
      pointerX + snapX.delta,
      west,
      event.altKey,
      MIN_SIZE_FRACTION * widthMM,
      widthMM,
    );
    let [minY, maxY] = resizeAxis(
      [start.minY, start.maxY],
      pointerY + snapY.delta,
      north,
      event.altKey,
      MIN_SIZE_FRACTION * heightMM,
      heightMM,
    );

    const ratio = slotRatio(start, event.shiftKey);
    if (ratio !== null) {
      const widthChange =
        Math.abs(maxX - minX - (start.maxX - start.minX)) / (start.maxX - start.minX);
      const heightChange =
        Math.abs(maxY - minY - (start.maxY - start.minY)) / (start.maxY - start.minY);
      if (widthChange >= heightChange) {
        [minY, maxY] = fitAxis(
          [start.minY, start.maxY],
          (maxX - minX) / ratio,
          north,
          event.altKey,
          heightMM,
        );
      } else {
        [minX, maxX] = fitAxis(
          [start.minX, start.maxX],
          (maxY - minY) * ratio,
          west,
          event.altKey,
          widthMM,
        );
      }
    }

    slot.x = minX / widthMM;
    slot.y = minY / heightMM;
    slot.w = (maxX - minX) / widthMM;
    slot.h = (maxY - minY) / heightMM;
    feedback.value = axisFeedback(snapX.line, ratio === null ? snapY.line : null);
  }

  /** slotRatio returns the width to height ratio a resize keeps, the locked one or with Shift the starting one. */
  function slotRatio(start: Bounds, shift: boolean): number | null {
    if (editor.slotRatioLock.locked.value) return editor.slotRatioLock.ratio.value;
    if (!shift) return null;
    const height = start.maxY - start.minY;
    return height > 0 ? (start.maxX - start.minX) / height : null;
  }

  /** finish ends a gesture, puts the moved slots on whole millimeters, and records the change. */
  function finish(): void {
    if (dragState) {
      const indices =
        dragState.mode === 'move' ? [...dragState.initialSlots.keys()] : [dragState.index];
      for (const index of indices) {
        const slot = activeSlots.value[index];
        if (slot) snapSlotToWholeMm(slot, state.paper, MIN_SIZE_FRACTION);
      }
    }
    mode.value = null;
    feedback.value = noSnap();
    dragState = null;
    editor.dragging.value = false;
    editor.history.commit();
  }

  /**
   * startMove selects a slot and starts moving the selection. Shift, Ctrl, or Cmd toggle the slot
   * in the selection. Pressing on an already selected slot keeps the selection, so a group moves together.
   * With Alt, the move takes copies of the selection along and leaves the originals in place.
   * Locked slots are selected but stay in place.
   */
  function startMove(event: PointerEvent, index: number): void {
    event.preventDefault();

    editor.selection.press(index, isAdditiveSelect(event));
    if (!editor.selection.has(index)) return;
    if (event.altKey) form.duplicateInPlace();

    const initialSlots = new Map<number, LayoutSlotRequest>();
    for (const selected of editor.selection.selected.value) {
      const slot = activeSlots.value[selected];
      if (slot && !slot.locked) initialSlots.set(selected, structuredClone(toRaw(slot)));
    }
    if (initialSlots.size === 0) return;

    dragState = {
      mode: 'move',
      initialSlots,
      start: options.toPaper(event.clientX, event.clientY),
      // The moved slots cannot snap onto themselves
      targets: targetsExcluding(new Set(initialSlots.keys())),
    };
    start('move', event);
  }

  /** startResize selects a slot and starts resizing it from a corner. */
  function startResize(event: PointerEvent, index: number, corner: SlotCorner): void {
    event.preventDefault();
    const slot = activeSlots.value[index];
    if (!slot || slot.locked) return;

    editor.selection.selectOnly(index);
    dragState = {
      mode: 'resize',
      index,
      corner,
      initial: slotBounds(slot, state.paper),
      start: options.toPaper(event.clientX, event.clientY),
      targets: targetsExcluding(new Set([index])),
    };
    start('resize', event);
  }

  /** start begins a gesture, during which the shortcuts and the automatic history checkpoint wait. */
  function start(next: DragMode, event: PointerEvent): void {
    mode.value = next;
    editor.dragging.value = true;
    pointerDrag.start(event);
  }

  return {
    mode,
    feedback,
    startMove,
    startResize,
  };
}

/**
 * resizeAxis returns the new start and end of a slot along one axis when one edge moves to a position.
 * The other edge stays, or mirrors the moving one around the center. The size keeps its minimum
 * and the slot stays within the paper.
 */
function resizeAxis(
  [start, end]: [number, number],
  edge: number,
  movesStart: boolean,
  fromCenter: boolean,
  minimum: number,
  length: number,
): [number, number] {
  if (fromCenter) {
    const center = (start + end) / 2;
    const half = Math.max(minimum / 2, movesStart ? center - edge : edge - center);
    const reach = Math.min(half, center, length - center);
    return [center - reach, center + reach];
  }
  if (movesStart) return [clamp(edge, 0, end - minimum), end];
  return [start, clamp(edge, start + minimum, length)];
}

/**
 * fitAxis returns the start and end of a slot along one axis at the size a ratio asks for. The edge opposite
 * the dragged corner stays, or the center with Alt, and the slot stays within the paper.
 */
function fitAxis(
  [start, end]: [number, number],
  size: number,
  movesStart: boolean,
  fromCenter: boolean,
  length: number,
): [number, number] {
  if (fromCenter) {
    const center = (start + end) / 2;
    const half = Math.min(size / 2, center, length - center);
    return [center - half, center + half];
  }
  if (movesStart) return [Math.max(0, end - size), end];
  return [start, Math.min(length, start + size)];
}
