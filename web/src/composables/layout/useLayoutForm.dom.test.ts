import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { toRaw } from 'vue';

import { ApiError } from '@/api/client/errors';
import type { LayoutDetailResponse, LayoutResponse } from '@/api/types/layout';
import { LAYOUT_LIMITS, type LayoutForm } from '@/composables/layout/useLayoutForm';
import { slotBounds } from '@/lib/layout/layoutGeometry';
import { layoutFormMocks, mountLayoutForm, resetLayoutFormMocks } from '@/test/layoutForm';

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
    te: () => false,
  }),
}));
vi.mock('@nuxt/ui/composables', async () => {
  const mocks = (await import('@/test/layoutFormMocks')).layoutFormMocks;
  return {
    useToast: () => mocks.toast,
  };
});
vi.mock('@/composables/ui/useConfirm', async () => {
  const mocks = (await import('@/test/layoutFormMocks')).layoutFormMocks;
  return {
    useConfirm: () => ({
      confirm: mocks.confirm,
    }),
  };
});
vi.mock('@/api/repositories/layout', async () => ({
  layoutRepository: (await import('@/test/layoutFormMocks')).layoutFormMocks.layouts,
}));
vi.mock('@/lib/browser/download', async () => ({
  downloadJSON: (await import('@/test/layoutFormMocks')).layoutFormMocks.download,
}));
vi.mock('@/lib/layout/sampleImages', async () => ({
  sampleImageStore: (await import('@/test/layoutFormMocks')).layoutFormMocks.images,
}));

const { images, layouts, toast, confirm, download } = layoutFormMocks;

/** Lets pending promise callbacks run. */
async function flush(): Promise<void> {
  for (let i = 0; i < 5; i++) await Promise.resolve();
}

/** Returns a valid layout with two pages. */
function storedLayout(): LayoutDetailResponse {
  return {
    name: 'Two pages',
    description: null,
    paper: {
      widthMM: 100,
      heightMM: 150,
    },
    pages: [
      {
        slots: [
          {
            x: 0,
            y: 0,
            w: 0.5,
            h: 0.5,
            locked: false,
          },
          {
            x: 0.5,
            y: 0.5,
            w: 0.5,
            h: 0.5,
            locked: false,
          },
        ],
      },
      {
        slots: [
          {
            x: 0,
            y: 0,
            w: 1,
            h: 1,
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

/** Returns the summary of the stored layout of storedLayout. */
function storedSummary(): LayoutResponse {
  return {
    name: 'Two pages',
    description: null,
    paperWidthMM: 100,
    paperHeightMM: 150,
    slots: 3,
    pages: 2,
    inUse: false,
  };
}

/** Returns the names of the fields that fail validation. */
function failedFields(form: LayoutForm): (string | undefined)[] {
  return form.validate(form.state).map((error) => error.name);
}

describe('provideLayoutForm', () => {
  let unmount = () => {};

  beforeEach(() => {
    localStorage.clear();
    resetLayoutFormMocks();
  });
  afterEach(() => unmount());

  /** Creates the layout form inside a mounted component. */
  function mountForm(): LayoutForm {
    const mounted = mountLayoutForm((layoutForm) => layoutForm);
    unmount = mounted.unmount;
    return mounted.layoutForm;
  }

  describe('hasEmptyPage', () => {
    it('reports a page without slots', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());

      form.removeSlots([0, 1]);

      expect(form.hasEmptyPage.value).toBe(true);
    });
  });

  describe('applyLayout', () => {
    it('shows a stored layout without changes and selects its first slot', () => {
      const form = mountForm();

      form.applyLayout(storedLayout());

      expect(form.isDirty.value).toBe(false);
      expect(form.isDraft.value).toBe(false);
      expect(form.editor.selection.active.value).toBe(0);
    });

    it('shows the background image kept for the layout', () => {
      const form = mountForm();

      form.applyLayout(storedLayout());

      expect(images.get).toHaveBeenCalledWith('layout:Two pages');
    });
  });

  describe('openImported', () => {
    it('shows the layout of a file as a new layout', () => {
      const form = mountForm();

      form.openImported(storedLayout());

      expect(form.isDraft.value).toBe(true);
      expect(form.state.name).toBe('Two pages');
      expect(form.editor.selection.active.value).toBe(0);
    });
  });

  describe('loadLayouts', () => {
    it('stores the stored layouts', async () => {
      const form = mountForm();
      layouts.list.mockResolvedValue([storedSummary()]);

      await form.loadLayouts();

      expect(form.layouts.value.map((layout) => layout.name)).toEqual(['Two pages']);
      expect(form.loadingList.value).toBe(false);
    });

    it('reports a failure', async () => {
      const form = mountForm();
      layouts.list.mockRejectedValue(new Error('offline'));

      await form.loadLayouts();

      expect(toast.add).toHaveBeenCalledOnce();
    });
  });

  describe('selectLayout', () => {
    it('opens a stored layout as the saved state', async () => {
      const form = mountForm();
      layouts.get.mockResolvedValue(storedLayout());

      await form.selectLayout('Two pages');

      expect(form.savedName.value).toBe('Two pages');
      expect(form.isDirty.value).toBe(false);
      expect(form.openCount.value).toBe(1);
    });

    it('keeps an opened layout when it is selected again', async () => {
      const form = mountForm();
      form.applyLayout(storedLayout());

      await form.selectLayout('Two pages');

      expect(layouts.get).not.toHaveBeenCalled();
    });

    it('opens only the latest selected layout when an earlier answer arrives late', async () => {
      const form = mountForm();
      let resolve: (layout: LayoutDetailResponse) => void = () => {};
      layouts.get.mockReturnValueOnce(
        new Promise((done) => {
          resolve = done;
        }),
      );
      layouts.get.mockResolvedValueOnce({
        ...storedLayout(),
        name: 'second',
      });
      const first = form.selectLayout('first');
      await form.selectLayout('second');

      resolve({
        ...storedLayout(),
        name: 'first',
      });
      await first;

      expect(form.savedName.value).toBe('second');
    });

    it('offers the unsaved changes the browser kept for the layout', async () => {
      const form = mountForm();
      layouts.get.mockResolvedValue(storedLayout());
      localStorage.setItem(
        'layoutEditor.draft.Two pages',
        JSON.stringify({
          ...storedLayout(),
          name: 'Kept',
        }),
      );

      await form.selectLayout('Two pages');

      expect(form.state.name).toBe('Kept');
      expect(form.isDirty.value).toBe(true);
    });

    it('shows every hidden slot of the last layout again', async () => {
      const form = mountForm();
      form.applyLayout({
        ...storedLayout(),
        name: 'other',
      });
      form.editor.visibility.toggle(0);
      layouts.get.mockResolvedValue(storedLayout());

      await form.selectLayout('Two pages');

      expect(form.editor.visibility.isHidden(0)).toBe(false);
    });

    it('reports a layout that cannot be loaded', async () => {
      const form = mountForm();
      layouts.get.mockRejectedValue(new Error('missing'));

      await form.selectLayout('missing');

      expect(toast.add).toHaveBeenCalledOnce();
    });

    it('asks before unsaved changes are dropped', async () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.state.name = 'Changed';
      confirm.mockResolvedValue(false);

      await form.selectLayout('other');

      expect(layouts.get).not.toHaveBeenCalled();
      expect(form.state.name).toBe('Changed');
    });
  });

  describe('showFirstLayout', () => {
    it('opens the first stored layout', async () => {
      const form = mountForm();
      layouts.list.mockResolvedValue([storedSummary()]);
      layouts.get.mockResolvedValue(storedLayout());
      await form.loadLayouts();

      await form.showFirstLayout();

      expect(layouts.get).toHaveBeenCalledWith('Two pages');
    });

    it('starts a new layout without stored ones', async () => {
      const form = mountForm();

      await form.showFirstLayout();

      expect(form.isDraft.value).toBe(true);
    });
  });

  describe('newLayout', () => {
    it('starts a new layout', async () => {
      const form = mountForm();
      form.applyLayout(storedLayout());

      await form.newLayout();

      expect(form.isDraft.value).toBe(true);
      expect(form.savedName.value).toBeNull();
    });

    it('asks before unsaved changes are dropped', async () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.state.name = 'Changed';
      confirm.mockResolvedValue(false);

      await form.newLayout();

      expect(form.isDraft.value).toBe(false);
    });
  });

  describe('importLayout', () => {
    it('opens a layout file as a new layout named after the file without a name', async () => {
      const form = mountForm();
      const file = new File(
        [
          JSON.stringify({
            ...storedLayout(),
            name: '',
          }),
        ],
        'From file.json',
      );

      await form.importLayout(file);

      expect(form.isDraft.value).toBe(true);
      expect(form.state.name).toBe('From file');
    });

    it('asks before an import replaces a stored layout', async () => {
      const form = mountForm();
      layouts.list.mockResolvedValue([storedSummary()]);
      await form.loadLayouts();
      confirm.mockResolvedValue(false);

      await form.importLayout(new File([JSON.stringify(storedLayout())], 'two-pages.json'));

      expect(confirm).toHaveBeenCalledOnce();
      expect(form.openCount.value).toBe(0);
    });

    it('rejects a file that is not a layout', async () => {
      const form = mountForm();

      await form.importLayout(new File(['not json'], 'notes.txt'));

      expect(toast.add).toHaveBeenCalledOnce();
    });

    it('ignores a missing file', async () => {
      const form = mountForm();

      await form.importLayout(undefined);

      expect(toast.add).not.toHaveBeenCalled();
    });
  });

  describe('applySaved', () => {
    it('keeps the background image of a new layout under its name', async () => {
      const form = mountForm();
      form.startNewLayout();
      const image = new Blob(['image']);
      await form.setSampleImage(image);

      await form.applySaved(
        {
          ...storedLayout(),
          name: 'saved',
        },
        null,
      );

      expect(images.set).toHaveBeenCalledWith('layout:saved', image);
    });
  });

  describe('setSampleImage', () => {
    it('keeps the image of a stored layout at once', async () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      const image = new Blob(['image']);

      await form.setSampleImage(image);

      expect(images.set).toHaveBeenCalledWith('layout:Two pages', image);
    });

    it('keeps the image of a new layout until its first save', async () => {
      const form = mountForm();
      form.startNewLayout();

      await form.setSampleImage(new Blob(['image']));

      expect(images.set).not.toHaveBeenCalled();
    });
  });

  describe('downloadLayout', () => {
    it('downloads the layout with its unsaved changes', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.state.description = 'Changed';

      form.downloadLayout();

      expect(download).toHaveBeenCalledWith(
        'Two pages.json',
        expect.objectContaining({
          name: 'Two pages',
          description: 'Changed',
        }),
      );
    });
  });

  describe('saveLayout', () => {
    it('saves the layout and reloads the stored layouts', async () => {
      const form = mountForm();
      form.startNewLayout();
      form.state.name = 'Saved';
      layouts.save.mockImplementation(async (_name, request) => ({
        ...storedLayout(),
        ...request,
      }));

      await form.saveLayout();
      await flush();

      expect(layouts.save).toHaveBeenCalledWith(
        'Saved',
        expect.objectContaining({
          name: 'Saved',
        }),
      );
      expect(form.savedName.value).toBe('Saved');
      expect(layouts.list).toHaveBeenCalled();
    });

    it('saves a renamed layout under its new name and deletes it under the old one', async () => {
      const form = mountForm();
      layouts.list.mockResolvedValue([storedSummary()]);
      await form.loadLayouts();
      form.applyLayout(storedLayout());
      form.state.name = 'Renamed';
      layouts.save.mockImplementation(async (_name, request) => ({
        ...storedLayout(),
        ...request,
      }));

      await form.saveLayout();
      await flush();

      expect(layouts.save).toHaveBeenCalledWith('Renamed', expect.anything());
      expect(layouts.delete).toHaveBeenCalledWith('Two pages');
      expect(form.savedName.value).toBe('Renamed');
    });

    it('rejects a new name another layout has', async () => {
      const form = mountForm();
      layouts.list.mockResolvedValue([
        storedSummary(),
        {
          ...storedSummary(),
          name: 'Taken',
        },
      ]);
      await form.loadLayouts();
      form.applyLayout(storedLayout());
      form.state.name = 'Taken';

      await form.saveLayout();

      expect(layouts.save).not.toHaveBeenCalled();
      expect(toast.add).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'pages.layouts.layoutExists',
        }),
      );
    });

    it('keeps the name of the layout the print settings use', async () => {
      const form = mountForm();
      layouts.list.mockResolvedValue([
        {
          ...storedSummary(),
          inUse: true,
        },
      ]);
      await form.loadLayouts();
      form.applyLayout(storedLayout());
      form.state.name = 'Renamed';

      await form.saveLayout();

      expect(layouts.save).not.toHaveBeenCalled();
      expect(toast.add).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'pages.layouts.renameInUse',
        }),
      );
    });
  });

  describe('deleteLayout', () => {
    it('deletes the stored layout after a confirmation and shows the first one', async () => {
      const form = mountForm();
      form.applyLayout(storedLayout());

      await form.deleteLayout();
      await flush();

      expect(layouts.delete).toHaveBeenCalledWith('Two pages');
      expect(images.delete).toHaveBeenCalledWith('layout:Two pages');
      expect(form.isDraft.value).toBe(true);
    });

    it('keeps the layout without a confirmation', async () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      confirm.mockResolvedValue(false);

      await form.deleteLayout();

      expect(layouts.delete).not.toHaveBeenCalled();
    });

    it('reports a layout the print settings use', async () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      layouts.delete.mockRejectedValue(
        new ApiError({
          message: 'in use',
          status: 409,
          code: 'conflict',
        }),
      );

      await form.deleteLayout();
      await flush();

      expect(toast.add).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'pages.layouts.deleteInUse',
        }),
      );
    });

    it('ignores a new layout', async () => {
      const form = mountForm();
      form.startNewLayout();

      await form.deleteLayout();

      expect(confirm).not.toHaveBeenCalled();
    });
  });

  describe('removeSlots', () => {
    it('selects the slot that takes the place of a removed one', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());

      form.removeSlots([0]);

      expect(form.activeSlots.value).toHaveLength(1);
      expect(form.editor.selection.active.value).toBe(0);
    });

    it('clears the selection after removing several slots', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());

      form.removeSlots([0, 1]);

      expect(form.editor.selection.selected.value.size).toBe(0);
    });
  });

  describe('removeSelectedSlots', () => {
    it('removes the selected slots', async () => {
      const form = mountForm();
      form.applyLayout(storedLayout());

      await form.removeSelectedSlots();

      expect(form.activeSlots.value).toHaveLength(1);
      expect(confirm).not.toHaveBeenCalled();
    });

    it('asks before the last slot of the page is removed', async () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.editor.selection.selectAll();
      confirm.mockResolvedValue(false);

      await form.removeSelectedSlots();

      expect(form.activeSlots.value).toHaveLength(2);
    });
  });

  describe('toggleLock', () => {
    it('locks the selected slots', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.editor.selection.selectAll();

      form.toggleLock();

      expect(form.activeSlots.value.map((slot) => slot.locked)).toEqual([true, true]);
    });

    it('records the lock as one history entry', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.editor.selection.selectAll();
      form.toggleLock();

      form.editor.history.undo();

      expect(form.activeSlots.value[0]!.locked).toBe(false);
    });

    it('unlocks the selection once every slot is locked', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.toggleLock();

      form.toggleLock();

      expect(form.activeSlots.value[0]!.locked).toBe(false);
    });
  });

  describe('setSlotLocked', () => {
    it('locks a single slot', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());

      form.setSlotLocked(1, true);

      expect(form.activeSlots.value.map((slot) => slot.locked)).toEqual([false, true]);
    });
  });

  describe('moveSlot', () => {
    it('moves a slot to another position and keeps it selected', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      const second = structuredClone(toRaw(form.activeSlots.value[1]!));

      form.moveSlot(1, 0);

      expect(form.activeSlots.value[0]).toEqual(second);
      expect([...form.editor.selection.selected.value]).toEqual([0]);
    });

    it('records the new order as one history entry', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      const first = structuredClone(toRaw(form.activeSlots.value[0]!));
      form.moveSlot(1, 0);

      form.editor.history.undo();

      expect(form.activeSlots.value[0]).toEqual(first);
    });

    it('shows every hidden slot again', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.editor.visibility.toggle(0);

      form.moveSlot(1, 0);

      expect(form.editor.visibility.isHidden(0)).toBe(false);
    });

    it('ignores a position outside the page', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      const before = structuredClone(toRaw(form.activeSlots.value));

      form.moveSlot(0, 5);

      expect(form.activeSlots.value).toEqual(before);
    });
  });

  describe('duplicateInPlace', () => {
    it('adds unlocked copies of the selection at their places and selects them', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.activeSlots.value[0]!.locked = true;
      form.editor.selection.selectOnly(0);

      form.duplicateInPlace();

      expect(form.activeSlots.value[2]).toEqual({
        ...form.activeSlots.value[0],
        locked: false,
      });
      expect([...form.editor.selection.selected.value]).toEqual([2]);
    });

    it('leaves the slots alone without a selection', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.editor.selection.clear();

      form.duplicateInPlace();

      expect(form.activeSlots.value).toHaveLength(2);
    });
  });

  describe('transformSelection', () => {
    it('moves and scales the selected slots into new bounds', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.editor.selection.selectAll();

      form.transformSelection({
        minX: 10,
        maxX: 60,
        minY: 0,
        maxY: 150,
      });

      expect(slotBounds(form.activeSlots.value[0]!, form.state.paper)).toMatchObject({
        minX: expect.closeTo(10),
        maxX: expect.closeTo(35),
      });
      expect(slotBounds(form.activeSlots.value[1]!, form.state.paper)).toMatchObject({
        minX: expect.closeTo(35),
        maxX: expect.closeTo(60),
      });
    });

    it('keeps a locked slot in place', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.activeSlots.value[1]!.locked = true;
      const before = structuredClone(toRaw(form.activeSlots.value[1]!));
      form.editor.selection.selectAll();

      form.transformSelection({
        minX: 10,
        maxX: 20,
        minY: 10,
        maxY: 20,
      });

      expect(form.activeSlots.value[1]).toEqual(before);
    });
  });

  describe('nudgeSlots', () => {
    it('moves the selected slots in whole millimeters', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      const { widthMM, heightMM } = form.state.paper;

      form.nudgeSlots(3, 2);

      const moved = form.activeSlots.value[0]!;
      expect(Math.round(moved.x * widthMM)).toBe(3);
      expect(Math.round(moved.y * heightMM)).toBe(2);
    });

    it('keeps the selected slots on the paper', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      const { widthMM, heightMM } = form.state.paper;
      form.nudgeSlots(3, 2);

      form.nudgeSlots(-widthMM, -heightMM);

      const moved = form.activeSlots.value[0]!;
      expect(moved.x).toBe(0);
      expect(moved.y).toBe(0);
    });

    it('keeps a locked slot in place', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.toggleLock();

      form.nudgeSlots(3, 2);

      expect(form.activeSlots.value[0]!.x).toBe(0);
    });

    it('leaves the slots alone without a selection', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.editor.selection.clear();
      const before = structuredClone(toRaw(form.activeSlots.value));

      form.nudgeSlots(1, 1);

      expect(form.activeSlots.value).toEqual(before);
    });
  });

  describe('alignSelection', () => {
    it('lines up the selected slots at an edge', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.editor.selection.selectAll();

      form.alignSelection('right');

      expect(form.activeSlots.value[0]!.x).toBe(0.5);
    });

    it('records the alignment as one history entry', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.editor.selection.selectAll();
      form.alignSelection('right');

      form.editor.history.undo();

      expect(form.activeSlots.value[0]!.x).toBe(0);
    });

    it('centers a single slot on the paper', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());

      form.alignSelection('centerX');

      expect(form.activeSlots.value[0]!.x).toBe(0.25);
    });

    it('keeps a locked slot in place', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.activeSlots.value[0]!.locked = true;
      form.editor.selection.selectAll();

      form.alignSelection('right');

      expect(form.activeSlots.value[0]!.x).toBe(0);
    });
  });

  describe('distributeSelection', () => {
    it('evens out the gaps between the selected slots', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.addSlot();
      Object.assign(form.activeSlots.value[2]!, {
        x: 0.7,
        y: 0,
        w: 0.1,
        h: 0.1,
      });
      Object.assign(form.activeSlots.value[1]!, {
        x: 0.9,
        w: 0.1,
      });
      form.editor.selection.selectAll();

      form.distributeSelection('x');

      expect(slotBounds(form.activeSlots.value[2]!, form.state.paper).minX).toBeCloseTo(65);
    });
  });

  describe('copySlots', () => {
    it('keeps the selected slots for pasting', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.copySlots();
      form.setActivePage(1);

      form.pasteSlots();

      expect(form.activeSlots.value).toHaveLength(2);
    });
  });

  describe('pasteSlots', () => {
    it('adds shifted, unlocked copies and selects them', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.toggleLock();
      form.copySlots();

      form.pasteSlots();

      const pasted = form.activeSlots.value[2]!;
      expect(slotBounds(pasted, form.state.paper).minX).toBeCloseTo(5);
      expect(pasted.locked).toBe(false);
      expect([...form.editor.selection.selected.value]).toEqual([2]);
    });

    it('shifts the next copies further', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.copySlots();
      form.pasteSlots();

      form.pasteSlots();

      expect(slotBounds(form.activeSlots.value[3]!, form.state.paper).minX).toBeCloseTo(10);
    });

    it('ignores an empty clipboard', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());

      form.pasteSlots();

      expect(form.activeSlots.value).toHaveLength(2);
    });
  });

  describe('duplicateSlots', () => {
    it('adds shifted copies of the selected slots and selects them', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.editor.selection.selectAll();

      form.duplicateSlots();

      expect(form.activeSlots.value).toHaveLength(4);
      expect([...form.editor.selection.selected.value]).toEqual([2, 3]);
    });
  });

  describe('setActivePage', () => {
    it('shows the other page and clears the selection', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());

      form.setActivePage(1);

      expect(form.activeSlots.value).toHaveLength(1);
      expect(form.editor.selection.selected.value.size).toBe(0);
    });
  });

  describe('addPage', () => {
    it('adds a page and shows it', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());

      form.addPage();

      expect(form.state.pages).toHaveLength(3);
      expect(form.activePageIndex.value).toBe(2);
    });
  });

  describe('removeActivePage', () => {
    it('removes the shown page and shows the page before it', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.addPage();

      form.removeActivePage();

      expect(form.state.pages).toHaveLength(2);
      expect(form.activePageIndex.value).toBe(1);
    });

    it('keeps the last page', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.addPage();
      form.removeActivePage();
      form.removeActivePage();

      form.removeActivePage();

      expect(form.state.pages).toHaveLength(1);
    });
  });

  describe('brandingFontSizePt', () => {
    it('shows the default caption size while the layout sets none', () => {
      const form = mountForm();

      form.applyLayout(storedLayout());

      expect(form.brandingFontSizePt.value).toBe(14);
    });
  });

  describe('brandingColor', () => {
    it('shows the default caption color while the layout sets none', () => {
      const form = mountForm();

      form.applyLayout(storedLayout());

      expect(form.brandingColor.value).toBe('#000000');
    });
  });

  describe('validate', () => {
    it('accepts a stored layout', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());

      const failed = failedFields(form);

      expect(failed).toEqual([]);
    });

    it('requires the name', () => {
      const form = mountForm();
      form.startNewLayout();

      const failed = failedFields(form);

      expect(failed).toEqual(['name']);
    });

    it('rejects a name over the maximum length', () => {
      const form = mountForm();
      form.applyLayout(storedLayout());
      form.state.name = 'x'.repeat(LAYOUT_LIMITS.name + 1);

      const failed = failedFields(form);

      expect(failed).toEqual(['name']);
    });
  });
});
