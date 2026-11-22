import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { LayoutDetailResponse } from '@/api/types/layout';
import { noSnap, type AxisTargets } from '@/lib/editor/snap';
import { slotBounds } from '@/lib/layout/layoutGeometry';
import { mountLayoutForm, resetLayoutFormMocks } from '@/test/layoutForm';
import { pointer } from '@/test/pointer';
import { useSlotDrag } from '@/composables/layout/useSlotDrag';

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
    te: () => false,
  }),
}));
vi.mock('@nuxt/ui/composables', async () => {
  const { layoutFormMocks } = await import('@/test/layoutFormMocks');
  return {
    useToast: () => layoutFormMocks.toast,
  };
});
vi.mock('@/composables/ui/useConfirm', async () => {
  const { layoutFormMocks } = await import('@/test/layoutFormMocks');
  return {
    useConfirm: () => ({
      confirm: layoutFormMocks.confirm,
    }),
  };
});
vi.mock('@/api/repositories/layout', async () => ({
  layoutRepository: (await import('@/test/layoutFormMocks')).layoutFormMocks.layouts,
}));
vi.mock('@/lib/layout/sampleImages', async () => ({
  sampleImageStore: (await import('@/test/layoutFormMocks')).layoutFormMocks.images,
}));

/**
 * Returns a layout on 100 x 100 mm paper with slot 0 from 10 to 30 mm
 * and slot 1 from 50 to 70 mm on both axes.
 */
function twoSlots(): LayoutDetailResponse {
  return {
    name: 'Two slots',
    description: null,
    paper: {
      widthMM: 100,
      heightMM: 100,
    },
    pages: [
      {
        slots: [
          {
            x: 0.1,
            y: 0.1,
            w: 0.2,
            h: 0.2,
            locked: false,
          },
          {
            x: 0.5,
            y: 0.5,
            w: 0.2,
            h: 0.2,
            locked: false,
          },
        ],
      },
    ],
    branding: {
      enabled: false,
      template: '',
      x: 0.5,
      y: 0.9,
      rotationDeg: 0,
    },
  };
}

describe('useSlotDrag', () => {
  let unmount = () => {};

  beforeEach(() => resetLayoutFormMocks());
  // A test may end with the pointer still pressed, so the gesture is released before the next one
  afterEach(() => {
    window.dispatchEvent(pointer('pointerup'));
    unmount();
  });

  /** Creates the slot drag over the layout, with client pixels as millimeters times the scale. */
  function mount(
    snapping = false,
    scale = 1,
    guides: AxisTargets = {
      x: [],
      y: [],
    },
  ) {
    const mounted = mountLayoutForm((form) => {
      form.applyLayout(twoSlots());
      const drag = useSlotDrag(
        {
          snapping,
          toPaper: (clientX, clientY) => ({
            x: clientX / scale,
            y: clientY / scale,
          }),
          scale,
          guides: () => guides,
        },
        form,
      );
      return {
        form,
        drag,
      };
    });
    unmount = mounted.unmount;
    return mounted.result;
  }

  /** Moves the pressed pointer to a point. */
  function move(to: { x: number; y: number }, init: PointerEventInit = {}): void {
    window.dispatchEvent(
      pointer('pointermove', {
        clientX: to.x,
        clientY: to.y,
        ...init,
      }),
    );
  }

  /** Returns the bounds of a slot of the active page in millimeters. */
  function boundsOf(form: ReturnType<typeof mount>['form'], index: number) {
    return slotBounds(form.activeSlots.value[index]!, form.state.paper);
  }

  describe('startMove', () => {
    it('selects the slot and starts a move', () => {
      const { form, drag } = mount();

      drag.startMove(pointer('pointerdown'), 1);

      expect(drag.mode.value).toBe('move');
      expect([...form.editor.selection.selected.value]).toEqual([1]);
    });

    it('moves the slot by the distance of the pointer on whole millimeters', () => {
      const { form, drag } = mount();
      drag.startMove(pointer('pointerdown'), 0);
      move({
        x: 5.4,
        y: 10,
      });

      window.dispatchEvent(pointer('pointerup'));

      expect(boundsOf(form, 0).minX).toBeCloseTo(15);
      expect(boundsOf(form, 0).minY).toBeCloseTo(20);
    });

    it('records one history entry for the whole move', () => {
      const { form, drag } = mount();
      drag.startMove(pointer('pointerdown'), 0);
      move({
        x: 5,
        y: 0,
      });
      move({
        x: 10,
        y: 0,
      });
      window.dispatchEvent(pointer('pointerup'));

      form.editor.history.undo();

      expect(boundsOf(form, 0).minX).toBeCloseTo(10);
    });

    it('holds back the automatic history entry until the move ends', () => {
      const { form, drag } = mount();

      drag.startMove(pointer('pointerdown'), 0);
      const during = form.editor.dragging.value;
      window.dispatchEvent(pointer('pointerup'));

      expect(during).toBe(true);
      expect(form.editor.dragging.value).toBe(false);
    });

    it('moves the whole selection and skips locked slots', () => {
      const { form, drag } = mount();
      form.activeSlots.value[1]!.locked = true;
      form.editor.selection.selectAll();

      drag.startMove(pointer('pointerdown'), 0);
      move({
        x: 10,
        y: 0,
      });
      window.dispatchEvent(pointer('pointerup'));

      expect(boundsOf(form, 0).minX).toBeCloseTo(20);
      expect(boundsOf(form, 1).minX).toBeCloseTo(50);
    });

    it('stops the whole selection at the paper border', () => {
      const { form, drag } = mount();
      form.editor.selection.selectAll();

      drag.startMove(pointer('pointerdown'), 0);
      move({
        x: 500,
        y: 0,
      });
      window.dispatchEvent(pointer('pointerup'));

      expect(boundsOf(form, 1).maxX).toBeCloseTo(100);
      expect(boundsOf(form, 0).minX).toBeCloseTo(40);
    });

    it('adds a slot to the selection with Ctrl or Cmd', () => {
      const { form, drag } = mount();

      drag.startMove(
        pointer('pointerdown', {
          ctrlKey: true,
        }),
        1,
      );

      expect([...form.editor.selection.selected.value]).toEqual([0, 1]);
    });

    it('aligns the edges of the moved slot with the other slots and marks the crossing', () => {
      const { form, drag } = mount(true);
      drag.startMove(pointer('pointerdown'), 0);

      move({
        x: 38,
        y: 42,
      });

      expect(boundsOf(form, 0).minX).toBeCloseTo(50);
      expect(boundsOf(form, 0).minY).toBeCloseTo(50);
      expect(drag.feedback.value.point).toEqual({
        x: 50,
        y: 50,
      });
    });

    it('places freely while Ctrl or Cmd is held', () => {
      const { form, drag } = mount(true);
      drag.startMove(pointer('pointerdown'), 0);

      move(
        {
          x: 38,
          y: 42,
        },
        {
          ctrlKey: true,
        },
      );

      expect(boundsOf(form, 0).minX).toBeCloseTo(48);
      expect(drag.feedback.value.point).toBeNull();
      expect(drag.feedback.value.lines).toEqual([]);
    });

    it('aligns a moved slot with a guide', () => {
      const { form, drag } = mount(true, 1, {
        x: [43],
        y: [],
      });
      drag.startMove(pointer('pointerdown'), 0);

      move({
        x: 32,
        y: 0,
      });

      expect(boundsOf(form, 0).minX).toBeCloseTo(43);
    });

    it('keeps the move on one axis while Shift is held', () => {
      const { form, drag } = mount();
      drag.startMove(pointer('pointerdown'), 0);

      move(
        {
          x: 20,
          y: 3,
        },
        {
          shiftKey: true,
        },
      );

      expect(boundsOf(form, 0).minX).toBeCloseTo(30);
      expect(boundsOf(form, 0).minY).toBeCloseTo(10);
    });

    it('shows the distances to the neighbors', () => {
      const { drag } = mount();
      drag.startMove(pointer('pointerdown'), 0);

      move({
        x: 0,
        y: 40,
      });

      expect(drag.feedback.value.gaps).toContainEqual(
        expect.objectContaining({
          axis: 'x',
          from: expect.closeTo(30),
          to: 50,
        }),
      );
    });

    it('moves copies of the slots and leaves the originals while Alt is held', () => {
      const { form, drag } = mount();
      drag.startMove(
        pointer('pointerdown', {
          altKey: true,
        }),
        0,
      );
      move({
        x: 20,
        y: 0,
      });

      window.dispatchEvent(pointer('pointerup'));

      expect(form.activeSlots.value).toHaveLength(3);
      expect(boundsOf(form, 0).minX).toBeCloseTo(10);
      expect(boundsOf(form, 2).minX).toBeCloseTo(30);
      expect([...form.editor.selection.selected.value]).toEqual([2]);
    });

    it('keeps the snap reach the same on the screen when zoomed', () => {
      const { form, drag } = mount(true, 4);
      drag.startMove(pointer('pointerdown'), 0);

      // Within the snap reach in millimeters, but outside it on the screen
      move({
        x: 37 * 4,
        y: 0,
      });

      expect(boundsOf(form, 0).minX).toBeCloseTo(47);
    });

    it('clears the snap feedback on release', () => {
      const { drag } = mount(true);
      drag.startMove(pointer('pointerdown'), 0);
      move({
        x: 38,
        y: 42,
      });

      window.dispatchEvent(pointer('pointerup'));

      expect(drag.feedback.value).toEqual(noSnap());
    });

    it('selects a locked slot without moving it', () => {
      const { form, drag } = mount();
      form.activeSlots.value[1]!.locked = true;

      drag.startMove(pointer('pointerdown'), 1);

      expect(form.editor.selection.active.value).toBe(1);
      expect(drag.mode.value).toBeNull();
    });
  });

  describe('startResize', () => {
    it('moves the edges at the dragged corner and keeps the opposite ones', () => {
      const { form, drag } = mount();
      drag.startResize(pointer('pointerdown'), 0, 'nw');

      move({
        x: -5,
        y: -5,
      });
      window.dispatchEvent(pointer('pointerup'));

      expect(boundsOf(form, 0)).toMatchObject({
        minX: expect.closeTo(5),
        minY: expect.closeTo(5),
        maxX: expect.closeTo(30),
        maxY: expect.closeTo(30),
      });
      expect(drag.mode.value).toBeNull();
    });

    it('keeps the smallest size', () => {
      const { form, drag } = mount();
      drag.startResize(pointer('pointerdown'), 0, 'se');

      move({
        x: -50,
        y: -50,
      });
      window.dispatchEvent(pointer('pointerup'));

      expect(boundsOf(form, 0).maxX - boundsOf(form, 0).minX).toBeCloseTo(5);
    });

    it('lets the height follow the width while the ratio is locked', () => {
      const { form, drag } = mount();
      form.editor.slotRatioLock.toggle(2);
      drag.startResize(pointer('pointerdown'), 0, 'se');

      move({
        x: 10,
        y: 0,
      });
      window.dispatchEvent(pointer('pointerup'));

      expect(boundsOf(form, 0).maxX - boundsOf(form, 0).minX).toBeCloseTo(30);
      expect(boundsOf(form, 0).maxY - boundsOf(form, 0).minY).toBeCloseTo(15);
    });

    it('keeps the ratio of the slot while Shift is held', () => {
      const { form, drag } = mount();
      drag.startResize(pointer('pointerdown'), 0, 'se');

      move(
        {
          x: 10,
          y: 0,
        },
        {
          shiftKey: true,
        },
      );

      expect(boundsOf(form, 0).maxX - boundsOf(form, 0).minX).toBeCloseTo(30);
      expect(boundsOf(form, 0).maxY - boundsOf(form, 0).minY).toBeCloseTo(30);
    });

    it('resizes from the center while Alt is held', () => {
      const { form, drag } = mount();
      drag.startResize(pointer('pointerdown'), 0, 'se');

      move(
        {
          x: 5,
          y: 5,
        },
        {
          altKey: true,
        },
      );

      expect(boundsOf(form, 0)).toMatchObject({
        minX: expect.closeTo(5),
        minY: expect.closeTo(5),
        maxX: expect.closeTo(35),
        maxY: expect.closeTo(35),
      });
    });

    it('aligns a moving edge with the other slots', () => {
      const { form, drag } = mount(true);
      drag.startResize(pointer('pointerdown'), 0, 'se');

      move({
        x: 18,
        y: 0,
      });

      expect(boundsOf(form, 0).maxX).toBeCloseTo(50);
      expect(drag.feedback.value.lines).toEqual([
        {
          axis: 'x',
          value: 50,
        },
      ]);
    });

    it('aligns a moving edge with a guide', () => {
      const { form, drag } = mount(true, 1, {
        x: [40],
        y: [],
      });
      drag.startResize(pointer('pointerdown'), 0, 'se');

      move({
        x: 8,
        y: 0,
      });

      expect(boundsOf(form, 0).maxX).toBeCloseTo(40);
    });

    it('snaps to no hidden slot', () => {
      const { form, drag } = mount(true);
      form.editor.visibility.toggle(1);
      drag.startResize(pointer('pointerdown'), 0, 'se');

      move({
        x: 38,
        y: 0,
      });

      expect(boundsOf(form, 0).maxX).toBeCloseTo(68);
    });

    it('ignores a locked slot', () => {
      const { form, drag } = mount();
      form.activeSlots.value[0]!.locked = true;

      drag.startResize(pointer('pointerdown'), 0, 'se');

      expect(drag.mode.value).toBeNull();
    });
  });
});
