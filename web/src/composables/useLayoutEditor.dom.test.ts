import { nextTick, reactive } from 'vue';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { LayoutDetailResponse } from '@/api/types';
import type { KeyValueStore } from '@/lib/keyValueStore';

import { useLayoutEditor } from '@/composables/useLayoutEditor';

const store = vi.hoisted(() => ({
  get: vi.fn<KeyValueStore<Blob>['get']>(async () => null),
  set: vi.fn<KeyValueStore<Blob>['set']>(async () => {}),
  delete: vi.fn<KeyValueStore<Blob>['delete']>(async () => {}),
}));

vi.mock('@/lib/sampleImages', () => ({
  sampleImageStore: store,
}));

/** layout returns a reactive layout with one page and one slot. */
function layout(): LayoutDetailResponse {
  return reactive<LayoutDetailResponse>({
    id: 'layout-1',
    name: 'Layout',
    paper: {
      widthMM: 100,
      heightMM: 150,
    },
    pages: [
      {
        slots: [
          {
            x: 0.1,
            y: 0.1,
            w: 0.8,
            h: 0.3,
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
  });
}

/** typeName changes the name of a layout twice in quick succession and lets the changes settle. */
async function typeName(state: LayoutDetailResponse): Promise<void> {
  state.name = 'Layout A';
  await nextTick();
  vi.advanceTimersByTime(100);
  state.name = 'Layout AB';
  await nextTick();
  vi.advanceTimersByTime(400);
}

describe('useLayoutEditor', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  /** setup creates the editor over a layout whose active page has three slots. */
  function setup() {
    const state = layout();
    const editor = useLayoutEditor(
      state,
      () => [0, 1, 2],
      () => state.id,
    );
    editor.history.reset();
    return {
      state,
      editor,
    };
  }

  describe('selection', () => {
    it('steps through the slots of the active page', () => {
      const { editor } = setup();
      editor.selection.selectOnly(2);

      editor.selection.selectNext(1);

      expect(editor.selection.active.value).toBe(0);
    });
  });

  describe('history', () => {
    it('reverts typed changes as one step', async () => {
      const { state, editor } = setup();
      await typeName(state);

      editor.history.undo();

      expect(state.name).toBe('Layout');
      expect(editor.history.canUndo.value).toBe(false);
    });

    it('leaves the commit of a drag to its end', async () => {
      const { state, editor } = setup();
      editor.dragging.value = true;

      state.pages[0]!.slots[0]!.x = 0.2;
      await nextTick();
      vi.advanceTimersByTime(1000);

      expect(editor.history.canUndo.value).toBe(false);
    });
  });

  describe('sampleImage', () => {
    it('keeps the photo of a layout under its ID', async () => {
      const { editor } = setup();
      const image = new Blob(['image']);

      await editor.sampleImage.set(image, 'layout-1');

      expect(store.set).toHaveBeenCalledWith('layout:layout-1', image);
    });

    it('never warns about the aspect ratio, since every slot crops the photo', () => {
      const { editor } = setup();

      editor.sampleImage.onLoad({
        target: {
          naturalWidth: 300,
          naturalHeight: 200,
        },
      } as unknown as Event);

      expect(editor.sampleImage.mismatch.value).toBe(false);
    });
  });

  describe('guides', () => {
    it('keeps the guides of a layout under its name', async () => {
      const { editor } = setup();
      const lines = [
        {
          axis: 'x' as const,
          value: 100,
        },
      ];

      editor.guides.lines.value = lines;
      await nextTick();

      expect(JSON.parse(localStorage.getItem('layoutEditor.guides.layout-1') ?? '[]')).toEqual(
        lines,
      );
    });
  });

  describe('slotRatioLock', () => {
    it('captures the ratio when switched on', () => {
      const editor = useLayoutEditor(
        layout(),
        () => [],
        () => null,
      );

      editor.slotRatioLock.toggle(0.75);

      expect(editor.slotRatioLock.locked.value).toBe(true);
      expect(editor.slotRatioLock.ratio.value).toBe(0.75);
    });

    it('keeps the captured ratio when switched off', () => {
      const editor = useLayoutEditor(
        layout(),
        () => [],
        () => null,
      );
      editor.slotRatioLock.toggle(0.75);

      editor.slotRatioLock.toggle(3);

      expect(editor.slotRatioLock.locked.value).toBe(false);
      expect(editor.slotRatioLock.ratio.value).toBe(0.75);
    });
  });
});
