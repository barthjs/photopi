import { ref, toRaw, toValue, type MaybeRefOrGetter } from 'vue';

import type { LayoutSlotRequest } from '@/api/types/layout';
import { isAdditiveSelect } from '@/composables/editor/useEditorSelection';
import { MIN_SIZE_FRACTION, type LayoutForm } from '@/composables/layout/useLayoutForm';
import { usePointerDrag } from '@/composables/editor/usePointerDrag';
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
import { paperSize, slotBounds, snapSlotToWholeMm } from '@/lib/layout/layoutGeometry';

/** The corner of a slot a resize handle sits on. */
export type SlotCorner = 'nw' | 'ne' | 'sw' | 'se';

type SnapTargets = {
  axes: AxisTargets;
  // The bounds of the slots that stay, which the distances are measured to
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
      // The slot at the start of the gesture in millimeters
      initial: Bounds;
      start: Point;
      targets: SnapTargets;
    };

export type DragMode = DragState['mode'];

export type SlotDragOptions = {
  snapping: MaybeRefOrGetter<boolean>;
  // Converts a position in the browser window into millimeters on the paper
  toPaper: (clientX: number, clientY: number) => Point;
  // The screen pixels per millimeter, so the snap reach stays the same on screen
  scale: MaybeRefOrGetter<number>;
  // The guides the slots snap to, in millimeters
  guides: () => AxisTargets;
};

/**
 * Moves slots and resizes a slot from its corners with the pointer in millimeters on the paper, one history entry per gesture,
 * with snapping that Ctrl or Cmd suspends, Shift keeping an axis or the ratio, and Alt duplicating or resizing from the center.
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
   * Collects the lines of the paper, of the guides,
   * and of every slot except the dragged ones and the hidden ones.
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

  /** Returns the snap reach in millimeters, or none while snapping is off or Ctrl or Cmd is held. */
  function snapDistance(event: PointerEvent): number {
    if (!toValue(options.snapping) || event.ctrlKey || event.metaKey) return 0;
    const scale = toValue(options.scale);
    return scale > 0 ? SNAP_DISTANCE / scale : SNAP_DISTANCE;
  }

  /** Applies the distance the pointer traveled since the start of the gesture. */
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

  /** Moves the dragged slots from their start by the pointer distance as a group that stops at the paper border. */
  function moveSlots(
    drag: Extract<DragState, { mode: 'move' }>,
    pointerOffset: Point,
    event: PointerEvent,
  ): void {
    // The edges and the center of the group align with the lines in reach, an axis that aligns with nothing
    // keeps even gaps to the neighbors, and a move kept on one axis only snaps along it.
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
   * Moves the edges at the dragged corner by the pointer distance, keeping or with Alt mirroring the opposite edges.
   */
  function resizeSlot(
    drag: Extract<DragState, { mode: 'resize' }>,
    offset: Point,
    event: PointerEvent,
  ): void {
    // The moving edges align with the lines in reach. With Shift or a locked ratio, the other side follows
    // the side the pointer changed more, and staying on the paper wins over the exact ratio.
    const slot = activeSlots.value[drag.index];
    if (!slot) return;

    const { widthMM, heightMM } = state.paper;
    const initial = drag.initial;
    const west = drag.corner === 'nw' || drag.corner === 'sw';
    const north = drag.corner === 'nw' || drag.corner === 'ne';
    const reach = snapDistance(event);

    const pointerX = (west ? initial.minX : initial.maxX) + offset.x;
    const pointerY = (north ? initial.minY : initial.maxY) + offset.y;
    const snapX = snapValue([pointerX], drag.targets.axes.x, reach);
    const snapY = snapValue([pointerY], drag.targets.axes.y, reach);

    let [minX, maxX] = resizeAxis(
      [initial.minX, initial.maxX],
      pointerX + snapX.delta,
      west,
      event.altKey,
      MIN_SIZE_FRACTION * widthMM,
      widthMM,
    );
    let [minY, maxY] = resizeAxis(
      [initial.minY, initial.maxY],
      pointerY + snapY.delta,
      north,
      event.altKey,
      MIN_SIZE_FRACTION * heightMM,
      heightMM,
    );

    const ratio = slotRatio(initial, event.shiftKey);
    if (ratio !== null) {
      const widthChange =
        Math.abs(maxX - minX - (initial.maxX - initial.minX)) / (initial.maxX - initial.minX);
      const heightChange =
        Math.abs(maxY - minY - (initial.maxY - initial.minY)) / (initial.maxY - initial.minY);
      if (widthChange >= heightChange) {
        [minY, maxY] = fitAxis(
          [initial.minY, initial.maxY],
          (maxX - minX) / ratio,
          north,
          event.altKey,
          heightMM,
        );
      } else {
        [minX, maxX] = fitAxis(
          [initial.minX, initial.maxX],
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

  /** Returns the width to height ratio a resize keeps, the locked one or with Shift the starting one. */
  function slotRatio(bounds: Bounds, shift: boolean): number | null {
    if (editor.slotRatioLock.locked.value) return editor.slotRatioLock.ratio.value;
    if (!shift) return null;
    const height = bounds.maxY - bounds.minY;
    return height > 0 ? (bounds.maxX - bounds.minX) / height : null;
  }

  /** Ends a gesture, puts the moved slots on whole millimeters, and records the change. */
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
   * Selects a slot, or toggles it with a modifier, and starts moving the unlocked selection,
   * or with Alt copies of it, while pressing a selected slot keeps the selection.
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

  /** Selects a slot and starts resizing it from a corner. */
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

  /** Begins a gesture, during which the shortcuts and the automatic history checkpoint wait. */
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
 * Returns the new start and end of a slot along one axis when one edge moves to a position,
 * keeping its minimum size and the slot on the paper.
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
 * Returns the start and end of a slot along one axis at the size a ratio asks for.
 * The edge opposite the dragged corner stays, or the center with Alt, and the slot stays within the paper.
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
