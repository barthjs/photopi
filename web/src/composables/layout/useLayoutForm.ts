import {
  computed,
  inject,
  provide,
  reactive,
  ref,
  toRaw,
  toValue,
  type InjectionKey,
  type MaybeRefOrGetter,
} from 'vue';
import { useI18n } from 'vue-i18n';
import { useToast } from '@nuxt/ui/composables';

import { ApiError } from '@/api/client/errors';
import { layoutRepository } from '@/api/repositories/layout';
import type {
  LayoutDetailResponse,
  LayoutResponse,
  LayoutSaveRequest,
  LayoutSlotRequest,
} from '@/api/types/layout';
import { useApiError } from '@/composables/form/useApiError';
import { useConfirm } from '@/composables/ui/useConfirm';
import { useDeleteConfirm } from '@/composables/ui/useDeleteConfirm';
import { useDraftRecovery } from '@/composables/editor/useDraftRecovery';
import { useDiscardGuard } from '@/composables/form/useDiscardConfirm';
import { useDirtyForm } from '@/composables/form/useDirtyForm';
import { useFormValidation, type FormErrorTarget } from '@/composables/form/useFormValidation';
import { useLayoutEditor } from '@/composables/layout/useLayoutEditor';
import { useMutation } from '@/composables/form/useMutation';
import {
  alignOffsets,
  alignReference,
  distributeOffsets,
  type AlignEdge,
  type DistributeAxis,
} from '@/lib/editor/align';
import {
  clamp,
  clampOffset,
  mapPointBetween,
  sizeBounds,
  unionBounds,
  type Bounds,
  type Point,
} from '@/lib/editor/geometry';
import { downloadJSON } from '@/lib/browser/download';
import { parseLayout } from '@/lib/layout/layoutFile';
import { paperSize, slotBounds, snapSlotToWholeMm } from '@/lib/layout/layoutGeometry';
import type { FormRules } from '@/lib/form/validation';

/** The maximum lengths the server accepts for the text fields of a layout. */
export const LAYOUT_LIMITS = {
  name: 60,
  description: 1000,
  brandingTemplate: 1024,
} as const;

/** The smallest width or height of a slot, as a fraction of the page. */
export const MIN_SIZE_FRACTION = 0.05;

/**
 * The server accepts up to 500 mm, a larger page would need a canvas too big for a low-power device.
 * It only requires more than zero, the editor works in whole millimeters.
 */
export const PAPER_MIN_MM = 1;
export const PAPER_MAX_MM = 500;

/** How far pasted and duplicated slots lie from their originals, so they do not cover them. */
const PASTE_OFFSET_MM = 5;

/** The size and color the server prints a caption with when the layout sets none. */
const DEFAULT_BRANDING_FONT_SIZE_PT = 14;
const DEFAULT_BRANDING_COLOR = '#000000';

export type LayoutForm = ReturnType<typeof createLayoutForm>;

const layoutFormKey: InjectionKey<LayoutForm> = Symbol('layoutForm');

/** Creates the layout form for a page and makes it available to its components. */
export function provideLayoutForm(
  form: MaybeRefOrGetter<FormErrorTarget | null | undefined>,
): LayoutForm {
  const layoutForm = createLayoutForm(form);
  provide(layoutFormKey, layoutForm);
  return layoutForm;
}

/** Returns the layout form provided by the layout page. */
export function useLayoutForm(): LayoutForm {
  const layoutForm = inject(layoutFormKey);
  if (!layoutForm) {
    throw new Error('useLayoutForm must be used below provideLayoutForm');
  }
  return layoutForm;
}

/**
 * Holds the edited layout, its active page, the editing state of the canvas,
 * and the operations on slots and stored layouts.
 */
function createLayoutForm(form: MaybeRefOrGetter<FormErrorTarget | null | undefined>) {
  const { t } = useI18n();
  const { messageFor } = useApiError();
  const { confirm } = useConfirm();
  const { confirmDelete } = useDeleteConfirm();
  const toast = useToast();

  const state = reactive<LayoutDetailResponse>(emptyLayout());
  const dirtyForm = useDirtyForm(state);
  const { confirmDiscard } = useDiscardGuard(dirtyForm.isDirty);

  const layouts = ref<LayoutResponse[]>([]);
  const loadingList = ref(false);
  // The name the edited layout is saved under, or null for a new, unsaved layout
  const savedName = ref<string | null>(null);
  const isDraft = computed(() => savedName.value === null);
  const activePageIndex = ref(0);
  // Counts every opened layout, also consecutive new ones,
  // so views can reset their editing aids whenever another layout opens.
  const openCount = ref(0);
  // Numbers the opened layouts and the requests for stored ones,
  // so a stored layout that arrives after another one was selected or opened does not replace it.
  let latestOpen = 0;

  // The unsaved changes of a stored layout survive a reload in the browser, opening it again offers them
  const recovery = useDraftRecovery({
    storageKey: 'layoutEditor',
    document: () => savedName.value,
    state,
    dirty: dirtyForm.isDirty,
    confirm: async () =>
      confirm({
        title: t('editor.draft.title'),
        description: t('editor.draft.description'),
        confirmLabel: t('editor.draft.restore'),
        icon: 'i-lucide-history',
      }),
  });

  // Reading gives the reactive slot array of the active page, so it can be changed in place.
  // Assigning replaces the slot array of that page.
  const activeSlots = computed<LayoutSlotRequest[]>({
    get: () => state.pages[activePageIndex.value]?.slots ?? [],
    set: (value) => {
      const page = state.pages[activePageIndex.value];
      if (page) page.slots = value;
    },
  });

  const editor = useLayoutEditor(
    state,
    () => [...activeSlots.value.keys()],
    () => savedName.value,
  );
  const { selection } = editor;

  // The position and size fields only apply to a single selected slot.
  // With several selected, they hide instead of showing the values of an arbitrary one.
  const selectedSlot = computed<LayoutSlotRequest | undefined>(() =>
    selection.active.value !== null ? activeSlots.value[selection.active.value] : undefined,
  );

  // The selected slots that can move, since a locked slot keeps its place
  const movableIndices = computed(() =>
    [...selection.selected.value].filter((index) => {
      const slot = activeSlots.value[index];
      return slot !== undefined && !slot.locked;
    }),
  );

  // The bounds of the selected slots in millimeters, which zooming to the selection fills the canvas with
  const selectionBounds = computed<Bounds | null>(() =>
    unionBounds(
      [...selection.selected.value]
        .map((index) => activeSlots.value[index])
        .filter((slot) => slot !== undefined)
        .map((slot) => slotBounds(slot, state.paper)),
    ),
  );

  // The slots copied last, kept for pasting into any page or layout
  const clipboard = ref<LayoutSlotRequest[]>([]);

  // The server rejects a page without slots, which the field rules cannot express
  const hasEmptyPage = computed(() => state.pages.some((page) => page.slots.length === 0));

  /** Starts a freshly shown layout with a clean editing state. */
  function startEditing(): void {
    latestOpen++;
    openCount.value++;
    toValue(form)?.clear();
    activePageIndex.value = 0;
    if (activeSlots.value.length) selection.selectOnly(0);
    else selection.clear();
    editor.visibility.showAll();
    dirtyForm.capture();
    editor.history.reset();
  }

  /** Shows a stored layout with the sample photo kept for it. */
  function applyLayout(layout: LayoutDetailResponse): void {
    Object.assign(state, structuredClone(layout));
    savedName.value = layout.name;
    startEditing();
    void editor.sampleImage.show(layout.name);
  }

  /** Shows a layout from a file as a new layout, which replaces a stored one with the same name on save. */
  function openImported(layout: LayoutDetailResponse): void {
    Object.assign(state, structuredClone(layout));
    savedName.value = null;
    startEditing();
  }

  /** Shows an empty layout to create. */
  function startNewLayout(): void {
    Object.assign(state, emptyLayout());
    savedName.value = null;
    startEditing();
  }

  /** Reads the stored layouts. */
  async function loadLayouts(): Promise<void> {
    loadingList.value = true;
    try {
      layouts.value = await layoutRepository.list();
    } catch (err) {
      toast.add({
        title: messageFor(err),
        icon: 'i-lucide-circle-alert',
        color: 'error',
      });
    } finally {
      loadingList.value = false;
    }
  }

  /**
   * Opens a stored layout after asking to discard unsaved changes.
   * Only the latest selection is opened, the answer for an earlier one is dropped.
   */
  async function selectLayout(name: string): Promise<void> {
    if (name === savedName.value) return;
    if (!(await confirmDiscard())) return;
    await openStored(name);
  }

  /** Opens a stored layout without asking, unless another one opened meanwhile. */
  async function openStored(name: string): Promise<void> {
    const request = ++latestOpen;
    try {
      const stored = await layoutRepository.get(name);
      if (request !== latestOpen) return;
      applyLayout(stored);
      await recovery.offer();
    } catch (err) {
      if (request !== latestOpen) return;
      toast.add({
        title: messageFor(err),
        icon: 'i-lucide-circle-alert',
        color: 'error',
      });
    }
  }

  /** Opens the first stored layout, or a new one if none is stored. */
  async function showFirstLayout(): Promise<void> {
    const first = layouts.value[0];
    if (first) {
      await openStored(first.name);
      return;
    }
    startNewLayout();
  }

  /** Starts a new layout after asking to discard unsaved changes. */
  async function newLayout(): Promise<void> {
    if (await confirmDiscard()) startNewLayout();
  }

  /**
   * Opens a layout file as a new layout, named after the file if it holds no name.
   * A stored layout with the same name is replaced on save, so the import asks first.
   */
  async function importLayout(file: File | null | undefined): Promise<void> {
    if (!file) return;

    const imported = parseLayout(await file.text());
    if (!imported) {
      toast.add({
        title: t('pages.layouts.importError', {
          name: file.name,
        }),
        icon: 'i-lucide-circle-alert',
        color: 'error',
      });
      return;
    }

    const name = imported.name.trim() || file.name.replace(/\.json$/i, '');

    if (layouts.value.some((stored) => stored.name === name)) {
      const replace = await confirm({
        title: t('pages.layouts.importReplaceTitle'),
        description: t('pages.layouts.importReplaceDescription', {
          name,
        }),
        confirmLabel: t('pages.layouts.importReplace'),
        color: 'error',
        icon: 'i-lucide-file-up',
      });
      if (!replace) return;
    }

    if (!(await confirmDiscard())) return;
    openImported({
      ...imported,
      name,
    });
  }

  /**
   * Shows the layout the server stored and tracks changes from it.
   * A new or renamed layout takes the sample photo and the guides along from its previous name.
   */
  async function applySaved(layout: LayoutDetailResponse, previous: string | null): Promise<void> {
    Object.assign(state, layout);
    savedName.value = layout.name;
    dirtyForm.capture();
    await editor.sampleImage.persist(layout.name, previous);
    editor.guides.persist(layout.name, previous);
  }

  /**
   * Shows a sample photo in the slots, or removes it with null, kept in the browser at once for a stored layout
   * and on the first save for a new one.
   */
  async function setSampleImage(image: Blob | null): Promise<void> {
    await editor.sampleImage.set(image, savedName.value);
  }

  /** Builds the request that saves the edited layout. */
  function saveRequest(): LayoutSaveRequest {
    return {
      name: state.name,
      description: state.description,
      paper: state.paper,
      pages: state.pages,
      branding: state.branding,
    };
  }

  /**
   * Saves the layout as shown in the editor, including unsaved changes,
   * as a file that openImported reads back.
   */
  function downloadLayout(): void {
    downloadJSON(`${state.name || 'layout'}.json`, saveRequest());
  }

  const { mutate: persistLayout, pending: saving } = useMutation(
    async () => layoutRepository.save(state.name, saveRequest()),
    {
      form,
      values: state,
      success: () => t('pages.layouts.saveSuccess'),
      onSuccess: async (saved) => {
        const previous = savedName.value;
        await applySaved(saved, previous);
        // A renamed layout leaves nothing under its old name
        if (previous !== null && previous !== saved.name) {
          recovery.discard(previous);
          await layoutRepository.delete(previous);
        }
        await loadLayouts();
      },
    },
  );

  /**
   * Stores the shown layout, a renamed one in place of the stored one,
   * unless the name is taken or the print settings use the stored one.
   */
  async function saveLayout(): Promise<void> {
    const previous = savedName.value;
    if (previous === null || previous === state.name) {
      await persistLayout();
      return;
    }

    if (layouts.value.some((stored) => stored.name === state.name)) {
      toast.add({
        title: t('pages.layouts.layoutExists'),
        icon: 'i-lucide-circle-alert',
        color: 'error',
      });
      return;
    }
    if (layouts.value.some((stored) => stored.name === previous && stored.inUse)) {
      toast.add({
        title: t('pages.layouts.renameInUse'),
        icon: 'i-lucide-circle-alert',
        color: 'error',
      });
      return;
    }
    await persistLayout();
  }

  const { mutate: removeLayout, pending: deleting } = useMutation(
    async (id: string) => layoutRepository.delete(id),
    {
      confirm: async () =>
        confirmDelete({
          title: t('pages.layouts.deleteConfirmTitle'),
          description: t('pages.layouts.deleteConfirmText'),
        }),
      // The server keeps the layout the print settings use
      onError: (err) => {
        if (!(err instanceof ApiError) || err.status !== 409) return false;
        toast.add({
          title: t('pages.layouts.deleteInUse'),
          icon: 'i-lucide-circle-alert',
          color: 'error',
        });
        return true;
      },
      onSuccess: async (_result, name) => {
        await editor.sampleImage.remove(name);
        editor.guides.remove(name);
        recovery.discard(name);
        await loadLayouts();
        await showFirstLayout();
      },
    },
  );

  /** Deletes the shown layout after a confirmation. */
  async function deleteLayout(): Promise<void> {
    const name = savedName.value;
    if (!name) return;

    await removeLayout(name);
  }

  /** Adds a slot to the active page and selects it. */
  function addSlot(): void {
    activeSlots.value.push(newSlot());
    selection.selectOnly(activeSlots.value.length - 1);
    editor.history.commit();
  }

  /** Removes slots of the active page and selects the slot that took the place of a single removed one. */
  function removeSlots(indices: number[]): void {
    if (indices.length === 0) return;
    const toRemove = new Set(indices);
    const anchorIndex = Math.min(...indices);
    // Filtering the raw array keeps reactive proxies out of the state, which the history clones
    activeSlots.value = toRaw(activeSlots.value).filter((_, i) => !toRemove.has(i));
    if (indices.length === 1 && activeSlots.value.length) {
      selection.selectOnly(Math.min(anchorIndex, activeSlots.value.length - 1));
    } else {
      selection.clear();
    }
    // The slots after a removed one move up, so the hidden ones would no longer match
    editor.visibility.showAll();
    editor.history.commit();
  }

  /** Removes the selected slots, after a confirmation if no slot would remain. */
  async function removeSelectedSlots(): Promise<void> {
    const indices = [...selection.selected.value];
    if (indices.length === 0) return;
    if (activeSlots.value.length - indices.length <= 0) {
      const confirmed = await confirmDelete({
        title: t('pages.layouts.removeLastSlotTitle'),
        description: t('pages.layouts.removeLastSlotConfirm'),
      });
      if (!confirmed) return;
    }
    removeSlots(indices);
  }

  /** Locks every selected slot while one of them is unlocked, and unlocks them otherwise. */
  function toggleLock(): void {
    const slots = [...selection.selected.value]
      .map((index) => activeSlots.value[index])
      .filter((slot) => slot !== undefined);
    if (slots.length === 0) return;

    const lock = slots.some((slot) => !slot.locked);
    for (const slot of slots) {
      slot.locked = lock;
    }
    editor.history.commit();
  }

  /** Locks or unlocks a single slot. */
  function setSlotLocked(index: number, locked: boolean): void {
    const slot = activeSlots.value[index];
    if (!slot || slot.locked === locked) return;
    slot.locked = locked;
    editor.history.commit();
  }

  /**
   * Moves a slot of the active page to another position, which changes the photo of the run it shows.
   * The moved slot stays selected, and every slot is shown again, since their positions change.
   */
  function moveSlot(from: number, to: number): void {
    const slots = [...toRaw(activeSlots.value)];
    const [moved] = slots.splice(from, 1);
    if (!moved || to < 0 || to > slots.length) return;
    slots.splice(to, 0, moved);
    activeSlots.value = slots;
    selection.set([to]);
    editor.visibility.showAll();
    editor.history.commit();
  }

  /**
   * Adds unlocked copies of the selected slots at their places and selects the copies,
   * so a drag can move them away from the originals. It records no history entry, the drag does.
   */
  function duplicateInPlace(): void {
    const copies = selectedSlots().map((slot) => ({
      ...slot,
      locked: false,
    }));
    if (copies.length === 0) return;

    const first = activeSlots.value.length;
    activeSlots.value.push(...copies);
    selection.set(copies.map((_, offset) => first + offset));
  }

  /**
   * Moves and scales the selected, unlocked slots together into new bounds in millimeters,
   * so they keep their places relative to each other. The slots stay on the paper and end on whole millimeters.
   */
  function transformSelection(target: Bounds): void {
    const indices = movableIndices.value;
    const from = unionBounds(
      indices.map((index) => slotBounds(activeSlots.value[index]!, state.paper)),
    );
    if (!from) return;

    const { widthMM, heightMM } = state.paper;
    for (const index of indices) {
      const slot = activeSlots.value[index]!;
      const bounds = slotBounds(slot, state.paper);
      const start = mapPointBetween(
        {
          x: bounds.minX,
          y: bounds.minY,
        },
        from,
        target,
      );
      const end = mapPointBetween(
        {
          x: bounds.maxX,
          y: bounds.maxY,
        },
        from,
        target,
      );
      slot.w = clamp((end.x - start.x) / widthMM, MIN_SIZE_FRACTION, 1);
      slot.h = clamp((end.y - start.y) / heightMM, MIN_SIZE_FRACTION, 1);
      slot.x = clamp(start.x / widthMM, 0, 1 - slot.w);
      slot.y = clamp(start.y / heightMM, 0, 1 - slot.h);
      snapSlotToWholeMm(slot, state.paper, MIN_SIZE_FRACTION);
    }
    editor.history.commit();
  }

  /**
   * Moves every selected, unlocked slot by whole millimeters within the paper.
   * The slots end on whole millimeters, like after a drag, and a series of nudges becomes one history entry once it rests.
   */
  function nudgeSlots(dxMm: number, dyMm: number): void {
    const dxFrac = dxMm / state.paper.widthMM;
    const dyFrac = dyMm / state.paper.heightMM;
    for (const index of movableIndices.value) {
      const slot = activeSlots.value[index]!;
      slot.x = clamp(slot.x + dxFrac, 0, 1 - slot.w);
      slot.y = clamp(slot.y + dyFrac, 0, 1 - slot.h);
      snapSlotToWholeMm(slot, state.paper, MIN_SIZE_FRACTION);
    }
  }

  /**
   * Lines up the selected, unlocked slots at an edge
   * or a center line of their bounds, or a single slot with the paper.
   */
  function alignSelection(edge: AlignEdge): void {
    const indices = movableIndices.value;
    if (indices.length === 0) return;

    const bounds = indices.map((index) => slotBounds(activeSlots.value[index]!, state.paper));
    moveSlots(indices, alignOffsets(bounds, edge, alignReference(bounds, paperSize(state.paper))));
    editor.history.commit();
  }

  /** Spaces the selected, unlocked slots evenly along an axis. */
  function distributeSelection(axis: DistributeAxis): void {
    const indices = movableIndices.value;
    const bounds = indices.map((index) => slotBounds(activeSlots.value[index]!, state.paper));
    moveSlots(indices, distributeOffsets(bounds, axis));
    editor.history.commit();
  }

  /** Moves each slot by its own shift in millimeters, kept on the paper. */
  function moveSlots(indices: number[], offsetsMm: Point[]): void {
    const paper = sizeBounds(paperSize(state.paper));
    indices.forEach((index, position) => {
      const slot = activeSlots.value[index]!;
      const offset = clampOffset(slotBounds(slot, state.paper), paper, offsetsMm[position]!);
      slot.x += offset.x / state.paper.widthMM;
      slot.y += offset.y / state.paper.heightMM;
    });
  }

  /** Keeps the selected slots for pasting. */
  function copySlots(): void {
    clipboard.value = selectedSlots();
  }

  /**
   * Adds the copied slots to the active page, shifted a little, and selects them.
   * Pasting again shifts the next copies further.
   */
  function pasteSlots(): void {
    if (clipboard.value.length === 0) return;
    clipboard.value = addCopies(clipboard.value);
  }

  /** Adds shifted copies of the selected slots to the active page and selects them. */
  function duplicateSlots(): void {
    addCopies(selectedSlots());
  }

  /** Returns plain copies of the selected slots of the active page. */
  function selectedSlots(): LayoutSlotRequest[] {
    return [...selection.selected.value]
      .map((index) => activeSlots.value[index])
      .filter((slot) => slot !== undefined)
      .map((slot) => structuredClone(toRaw(slot)));
  }

  /**
   * Adds unlocked copies of slots, shifted by PASTE_OFFSET_MM and kept on the paper,
   * selects them, and returns them.
   */
  function addCopies(slots: LayoutSlotRequest[]): LayoutSlotRequest[] {
    if (slots.length === 0) return [];

    const dx = PASTE_OFFSET_MM / state.paper.widthMM;
    const dy = PASTE_OFFSET_MM / state.paper.heightMM;
    const copies = slots.map((slot) => ({
      ...slot,
      x: clamp(slot.x + dx, 0, 1 - slot.w),
      y: clamp(slot.y + dy, 0, 1 - slot.h),
      locked: false,
    }));

    const first = activeSlots.value.length;
    activeSlots.value.push(...structuredClone(copies));
    selection.set(copies.map((_, offset) => first + offset));
    editor.history.commit();
    return copies;
  }

  // The selection holds slot positions within the active page, so every change of the active page clears it

  /** Shows another page. */
  function setActivePage(index: number): void {
    if (index === activePageIndex.value) return;
    activePageIndex.value = index;
    selection.clear();
    editor.visibility.showAll();
  }

  /** Adds a page with one slot and shows it. */
  function addPage(): void {
    state.pages.push({
      slots: [newSlot()],
    });
    activePageIndex.value = state.pages.length - 1;
    selection.clear();
    editor.history.commit();
  }

  /** Removes the shown page, unless it is the only one. */
  function removeActivePage(): void {
    if (state.pages.length <= 1) return;
    const index = activePageIndex.value;
    state.pages.splice(index, 1);
    activePageIndex.value = Math.min(index, state.pages.length - 1);
    selection.clear();
    editor.history.commit();
  }

  // The inputs need a value, while the layout leaves the font size and the color unset to use the defaults of the server.
  // Showing those defaults makes the preview match the print.
  const brandingFontSizePt = computed<number>({
    get: () => {
      const size = state.branding.fontSizePt;
      return size !== undefined && size > 0 ? size : DEFAULT_BRANDING_FONT_SIZE_PT;
    },
    set: (value) => {
      state.branding.fontSizePt = value;
    },
  });
  const brandingColor = computed<string>({
    get: () => state.branding.color || DEFAULT_BRANDING_COLOR,
    set: (value) => {
      state.branding.color = value;
    },
  });

  // The rules repeat the required fields and maximum lengths the server checks, with its tags
  const { validate } = useFormValidation<LayoutDetailResponse>(
    (): FormRules<LayoutDetailResponse> => ({
      name: {
        required: 'required',
        max: LAYOUT_LIMITS.name,
      },
      description: {
        max: LAYOUT_LIMITS.description,
      },
      'branding.template': {
        max: LAYOUT_LIMITS.brandingTemplate,
      },
    }),
  );

  return {
    state,
    editor,
    isDirty: dirtyForm.isDirty,
    fieldProps: dirtyForm.fieldProps,
    resetChanges: dirtyForm.reset,
    confirmDiscard,
    layouts,
    loadingList,
    openCount,
    savedName,
    isDraft,
    activePageIndex,
    activeSlots,
    selectedSlot,
    movableIndices,
    selectionBounds,
    hasEmptyPage,
    brandingFontSizePt,
    brandingColor,
    applyLayout,
    openImported,
    startNewLayout,
    loadLayouts,
    selectLayout,
    showFirstLayout,
    newLayout,
    importLayout,
    applySaved,
    setSampleImage,
    saveRequest,
    downloadLayout,
    saveLayout,
    saving,
    deleteLayout,
    deleting,
    addSlot,
    removeSlots,
    removeSelectedSlots,
    toggleLock,
    setSlotLocked,
    moveSlot,
    duplicateInPlace,
    transformSelection,
    nudgeSlots,
    alignSelection,
    distributeSelection,
    copySlots,
    pasteSlots,
    duplicateSlots,
    setActivePage,
    addPage,
    removeActivePage,
    validate,
  };
}

/** The layout a new one starts from. */
function emptyLayout(): LayoutDetailResponse {
  return {
    name: '',
    description: null,
    paper: {
      widthMM: 100,
      heightMM: 150,
    },
    pages: [
      {
        slots: [newSlot()],
      },
    ],
    branding: {
      enabled: false,
      template: '{{eventName}} · {{date}}',
      x: 0.5,
      y: 0.92,
      rotationDeg: 0,
    },
  };
}

/** Returns the slot a new page or the add button starts with. */
function newSlot(): LayoutSlotRequest {
  return {
    x: 0.1,
    y: 0.1,
    w: 0.3,
    h: 0.3,
    locked: false,
  };
}
