<script setup lang="ts">
import { computed, onMounted, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router';

import EditorSidebar from '@/components/editor/EditorSidebar.vue';
import EditorToolbar from '@/components/editor/EditorToolbar.vue';
import LayoutBrandingCard from '@/components/layout-editor/LayoutBrandingCard.vue';
import LayoutCanvas from '@/components/layout-editor/LayoutCanvas.vue';
import LayoutHelpModal from '@/components/layout-editor/LayoutHelpModal.vue';
import LayoutPageTabs from '@/components/layout-editor/LayoutPageTabs.vue';
import LayoutPaperCard from '@/components/layout-editor/LayoutPaperCard.vue';
import LayoutSlotCard from '@/components/layout-editor/LayoutSlotCard.vue';
import DocsLink from '@/components/system/DocsLink.vue';
import { useEditorShortcuts } from '@/composables/editor/useEditorShortcuts';
import { useEditorWorkspace } from '@/composables/editor/useEditorWorkspace';
import type { FormErrorTarget } from '@/composables/form/useFormValidation';
import { provideLayoutForm } from '@/composables/layout/useLayoutForm';
import { useSaveShortcut, type SubmitTarget } from '@/composables/form/useSaveShortcut';
import { useShortcutLabels } from '@/composables/ui/useShortcutLabels';
import { useUnloadGuard } from '@/composables/ui/useUnloadGuard';
import { shortcutKeys } from '@/lib/editor/shortcuts';
import { FORM_VALIDATE_ON } from '@/lib/form/validation';
import { LAYOUT_EDITOR_SHORTCUTS } from '@/lib/layout/layoutEditorShortcuts';
import { layoutMenuItems } from '@/lib/layout/layoutItems';
import { replaceRouteQuery } from '@/lib/ui/routeQuery';
import { shortcutDefinition } from '@/lib/ui/shortcuts';

// How far the arrow keys move the selected slots
const NUDGE_STEP_MM = 1;
const NUDGE_STEP_LARGE_MM = 10;

// The zoom factor of one click on the zoom buttons
const ZOOM_STEP = 1.25;

const { t } = useI18n();
const { keyLabels } = useShortcutLabels();
const route = useRoute();
const router = useRouter();

const form = useTemplateRef<FormErrorTarget & SubmitTarget>('form');
const canvas = useTemplateRef<InstanceType<typeof LayoutCanvas>>('canvas');
const layout = provideLayoutForm(form);
const { state, editor } = layout;
const { selection, history, sampleImage } = editor;

// The view mode shows the page as printed, with the sample photo or placeholders in the slots.
// A drag or a marquee keeps the keys from changing the slots under the pointer.
const workspace = useEditorWorkspace({
  storageKey: 'layoutEditor',
  onEnterViewMode: () => selection.clear(),
  onEscape: () => selection.clear(),
  onTab: (step) => selection.selectNext(step),
  busy: () => canvas.value?.dragging ?? false,
});
const { sidebarOpen, viewMode, helpOpen, fullscreen, whenEditing } = workspace;

// The hidden file input that picks the sample photo of the layout
const imageInput = useTemplateRef<HTMLInputElement>('imageInput');

// Another layout starts fitted, since the zoom and the pan of the last one rarely suit it
watch(
  () => layout.openCount.value,
  () => canvas.value?.resetZoom(),
);

// The URL keeps the name of the stored layout, so a reload shows it again
watch(
  () => layout.savedName.value,
  (name) => {
    void replaceRouteQuery(route, router, (query) => {
      const next = {
        ...query,
      };
      if (name) next.name = name;
      else delete next.name;
      return next;
    });
  },
);

/** Switches to another layout. */
function onSelectLayout(name: unknown): void {
  if (typeof name !== 'string' || !name) return;
  void layout.selectLayout(name);
}

onMounted(async () => {
  await layout.loadLayouts();
  const queryName = route.query.name;
  if (typeof queryName === 'string' && layout.layouts.value.some((l) => l.name === queryName)) {
    await layout.selectLayout(queryName);
  } else {
    await layout.showFirstLayout();
  }
});

const { kbds: saveKbds } = useSaveShortcut(
  form,
  () => !layout.hasEmptyPage.value && !layout.saving.value,
);

// Saving a draft creates the layout, so the button names that
const saveLabel = computed(() => (layout.isDraft.value ? t('common.create') : t('common.save')));

/** Shows the chosen file as the sample photo and keeps it for the layout. */
function onImageChange(event: Event): void {
  const input = event.target;
  if (!(input instanceof HTMLInputElement)) return;
  const file = input.files?.[0] ?? null;
  // Reset the input so choosing the same file again triggers another change
  input.value = '';
  if (file) void layout.setSampleImage(file);
}

// The digit keys select a slot of the active page
const slotShortcuts = Object.fromEntries(
  ['1', '2', '3', '4', '5', '6', '7', '8', '9'].map((digit) => [
    digit,
    whenEditing(() => selection.selectOnly(Number(digit) - 1)),
  ]),
);

useEditorShortcuts({
  workspace,
  history,
  canvas: () => canvas.value,
  selectAll: () => selection.selectAll(),
  pickImage: () => imageInput.value?.click(),
  nudge: layout.nudgeSlots,
  nudgeStep: NUDGE_STEP_MM,
  nudgeStepLarge: NUDGE_STEP_LARGE_MM,
  extra: {
    [shortcutDefinition(LAYOUT_EDITOR_SHORTCUTS.copy)]: whenEditing(() => layout.copySlots()),
    [shortcutDefinition(LAYOUT_EDITOR_SHORTCUTS.paste)]: whenEditing(() => layout.pasteSlots()),
    [shortcutDefinition(LAYOUT_EDITOR_SHORTCUTS.duplicate)]: whenEditing(() =>
      layout.duplicateSlots(),
    ),
    [shortcutDefinition(LAYOUT_EDITOR_SHORTCUTS.addSlot)]: whenEditing(() => layout.addSlot()),
    [shortcutDefinition(LAYOUT_EDITOR_SHORTCUTS.removeSlots)]: whenEditing(
      () => void layout.removeSelectedSlots(),
    ),
    [shortcutDefinition(LAYOUT_EDITOR_SHORTCUTS.removeSlotsAlternative)]: whenEditing(
      () => void layout.removeSelectedSlots(),
    ),
    [shortcutDefinition(LAYOUT_EDITOR_SHORTCUTS.lock)]: whenEditing(() => layout.toggleLock()),
    ...slotShortcuts,
  },
});

useUnloadGuard(() => layout.isDirty.value);

onBeforeRouteLeave(async () => layout.confirmDiscard());
</script>

<template>
  <!--
    From lg up, the admin layout gives the page the height of its panel, so the canvas size follows
    the layout and not the scroll position. Below lg, the canvas and the cards stack and the page scrolls.
  -->
  <UForm
    ref="form"
    :validate-on="FORM_VALIDATE_ON"
    :state="state"
    :validate="layout.validate"
    class="flex flex-col gap-3 lg:min-h-0 lg:flex-1"
    @submit="layout.saveLayout()"
  >
    <input ref="imageInput" type="file" accept="image/*" class="hidden" @change="onImageChange" />

    <!--
      The buttons share the row of the container, so on narrow screens the menu takes its own row,
      and the buttons below show only their icons.
    -->
    <div class="flex flex-wrap items-center gap-2">
      <div class="contents">
        <USelectMenu
          :model-value="layout.savedName.value ?? undefined"
          value-key="name"
          label-key="name"
          :items="layoutMenuItems(layout.layouts.value)"
          :loading="layout.loadingList.value"
          :placeholder="t('pages.layouts.selectPlaceholder')"
          :aria-label="t('pages.layouts.selectLabel')"
          class="w-full sm:w-56"
          @update:model-value="onSelectLayout"
        >
          <!-- A draft is not in the list yet, so the menu names it and marks it instead of showing the placeholder. -->
          <template v-if="layout.isDraft.value" #default>
            <span class="flex min-w-0 items-center gap-2">
              <span class="truncate">{{ state.name || t('pages.layouts.newLayout') }}</span>
              <UBadge :label="t('pages.layouts.draft')" color="info" size="sm" />
            </span>
          </template>
        </USelectMenu>
        <DocsLink page="layouts" />
        <UButton
          :label="t('pages.layouts.new')"
          :aria-label="t('pages.layouts.new')"
          icon="i-lucide-plus"
          color="neutral"
          variant="outline"
          :ui="{ label: 'hidden sm:inline' }"
          @click="layout.newLayout"
        />
        <UTooltip :text="t('common.delete')">
          <UButton
            icon="i-lucide-trash-2"
            color="error"
            variant="outline"
            :aria-label="t('common.delete')"
            :disabled="!layout.savedName.value"
            @click="layout.deleteLayout"
          />
        </UTooltip>

        <div class="bg-accented mx-1 hidden h-5 w-px sm:block"></div>

        <UFileUpload
          v-slot="{ open }"
          :model-value="null"
          reset
          accept=".json,application/json"
          :preview="false"
          :dropzone="false"
          @update:model-value="(file: File | null | undefined) => void layout.importLayout(file)"
          @reject="(files: File[]) => void layout.importLayout(files[0])"
        >
          <UButton
            :label="t('pages.layouts.import')"
            :aria-label="t('pages.layouts.import')"
            icon="i-lucide-file-up"
            color="neutral"
            variant="outline"
            :ui="{ label: 'hidden sm:inline' }"
            @click="open()"
          />
        </UFileUpload>
        <UButton
          :label="t('pages.layouts.download')"
          :aria-label="t('pages.layouts.download')"
          icon="i-lucide-file-down"
          color="neutral"
          variant="outline"
          :ui="{ label: 'hidden sm:inline' }"
          @click="layout.downloadLayout"
        />
      </div>
      <div class="ml-auto flex items-center gap-2">
        <UButton
          v-if="layout.isDirty.value"
          :label="t('common.reset')"
          :aria-label="t('common.reset')"
          icon="i-lucide-rotate-ccw"
          color="neutral"
          variant="outline"
          :ui="{ label: 'hidden sm:inline' }"
          @click="layout.resetChanges"
        />
        <UTooltip :text="saveLabel" :kbds="saveKbds">
          <UButton
            type="submit"
            :label="saveLabel"
            icon="i-lucide-save"
            :loading="layout.saving.value"
            :disabled="layout.hasEmptyPage.value"
          />
        </UTooltip>
      </div>
    </div>

    <!-- In fullscreen, the workspace covers the page, so the admin header and the navigation are out of the way. -->
    <div
      class="bg-default flex flex-col gap-3 lg:min-h-0 lg:flex-1 lg:flex-row"
      :class="{
        'fixed inset-0 z-50 overflow-y-auto p-3 lg:overflow-hidden': fullscreen.isFullscreen.value,
      }"
    >
      <div class="flex min-w-0 flex-col gap-2 lg:min-h-0 lg:flex-1">
        <EditorToolbar
          :workspace="workspace"
          :history="history"
          :zoom="canvas?.actualZoom ?? 1"
          :has-selection="selection.selected.value.size > 0"
          :align-count="layout.movableIndices.value.length"
          :image="sampleImage"
          :image-label="t('pages.layouts.sampleImage')"
          :guides="editor.guides"
          @zoom-in="canvas?.zoomBy(ZOOM_STEP)"
          @zoom-out="canvas?.zoomBy(1 / ZOOM_STEP)"
          @reset-zoom="canvas?.resetZoom()"
          @zoom-to-selection="canvas?.zoomToSelection()"
          @zoom-to="(factor) => canvas?.zoomToActual(factor)"
          @align="layout.alignSelection"
          @distribute="layout.distributeSelection"
          @pick-image="imageInput?.click()"
          @clear-image="void layout.setSampleImage(null)"
        >
          <UTooltip
            :text="t('pages.layouts.addSlot')"
            :kbds="keyLabels(shortcutKeys(LAYOUT_EDITOR_SHORTCUTS.addSlot))"
          >
            <UButton
              icon="i-lucide-square-plus"
              size="sm"
              color="neutral"
              variant="ghost"
              :aria-label="t('pages.layouts.addSlot')"
              :disabled="viewMode"
              @click="layout.addSlot"
            />
          </UTooltip>
        </EditorToolbar>

        <LayoutCanvas ref="canvas" :view="workspace.view" :view-mode="viewMode" />

        <LayoutPageTabs />
      </div>

      <!-- The cards stay mounted while hidden, so the form keeps its fields. -->
      <EditorSidebar v-show="sidebarOpen" storage-key="layoutEditor">
        <LayoutPaperCard />
        <LayoutSlotCard />
        <LayoutBrandingCard />
      </EditorSidebar>
    </div>

    <LayoutHelpModal v-model:open="helpOpen" />
  </UForm>
</template>
