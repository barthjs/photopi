<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import EditorItemList, { type EditorListItem } from '@/components/editor/EditorItemList.vue';
import EditorPanel from '@/components/editor/EditorPanel.vue';
import EditorTransformFields from '@/components/editor/EditorTransformFields.vue';
import { MIN_SIZE_FRACTION, useLayoutForm } from '@/composables/layout/useLayoutForm';
import { useShortcutLabels } from '@/composables/ui/useShortcutLabels';
import { clamp, unionBounds } from '@/lib/editor/geometry';
import { shortcutKeys } from '@/lib/editor/shortcuts';
import { LAYOUT_EDITOR_SHORTCUTS } from '@/lib/layout/layoutEditorShortcuts';
import { parseRatio, roundMm, slotBounds } from '@/lib/layout/layoutGeometry';

// Common width to height ratios of cameras and photos, to set the shape of a slot directly
const SLOT_RATIO_PRESETS = [
  {
    label: '1:1',
    ratio: 1,
  },
  {
    label: '4:3',
    ratio: 4 / 3,
  },
  {
    label: '3:4',
    ratio: 3 / 4,
  },
  {
    label: '3:2',
    ratio: 3 / 2,
  },
  {
    label: '2:3',
    ratio: 2 / 3,
  },
  {
    label: '16:9',
    ratio: 16 / 9,
  },
  {
    label: '9:16',
    ratio: 9 / 16,
  },
];

// The slots of the active page in their order, and the selected slot with its position and size in millimeters,
// or the position and size of several selected slots together.
const { t } = useI18n();
const { keyLabels } = useShortcutLabels();
const layout = useLayoutForm();
const { state, editor, activeSlots, selectedSlot, toggleLock } = layout;
const { selection, visibility } = editor;
const {
  locked: slotRatioLocked,
  toggle: toggleSlotRatioLock,
  heightFor: slotHeightFor,
  widthFor: slotWidthFor,
} = editor.slotRatioLock;

// The slots in the order the photos of a run fill them
const items = computed<EditorListItem<number>[]>(() =>
  activeSlots.value.map((slot, index) => ({
    key: index,
    label: t('pages.layouts.slotLabel', {
      n: index + 1,
    }),
    locked: slot.locked,
    hidden: visibility.isHidden(index),
  })),
);

// The bounds of the selected slots that can move, which the transform fields show
const movableBounds = computed(() =>
  unionBounds(
    layout.movableIndices.value.map((index) => slotBounds(activeSlots.value[index]!, state.paper)),
  ),
);

/** Selects a slot of the list, alone or in addition to the selection. */
function onSelect(index: number, additive: boolean): void {
  if (additive) {
    selection.toggle(index);
    return;
  }
  selection.selectOnly(index);
}

/** Locks or unlocks a slot of the list. */
function onToggleLock(index: number): void {
  const slot = activeSlots.value[index];
  if (slot) layout.setSlotLocked(index, !slot.locked);
}

const xMm = computed<number>({
  get: () => (selectedSlot.value ? roundMm(selectedSlot.value.x * state.paper.widthMM) : 0),
  set: (mm) => {
    if (!selectedSlot.value) return;
    selectedSlot.value.x = clamp(mm / state.paper.widthMM, 0, 1 - selectedSlot.value.w);
  },
});
const yMm = computed<number>({
  get: () => (selectedSlot.value ? roundMm(selectedSlot.value.y * state.paper.heightMM) : 0),
  set: (mm) => {
    if (!selectedSlot.value) return;
    selectedSlot.value.y = clamp(mm / state.paper.heightMM, 0, 1 - selectedSlot.value.h);
  },
});
const wMm = computed<number>({
  get: () => (selectedSlot.value ? roundMm(selectedSlot.value.w * state.paper.widthMM) : 0),
  set: (mm) => {
    const slot = selectedSlot.value;
    if (!slot) return;
    slot.w = clamp(mm / state.paper.widthMM, MIN_SIZE_FRACTION, 1 - slot.x);
    if (slotRatioLocked.value) {
      const hFrac = slotHeightFor(mm) / state.paper.heightMM;
      slot.h = clamp(hFrac, MIN_SIZE_FRACTION, 1 - slot.y);
    }
  },
});
const hMm = computed<number>({
  get: () => (selectedSlot.value ? roundMm(selectedSlot.value.h * state.paper.heightMM) : 0),
  set: (mm) => {
    const slot = selectedSlot.value;
    if (!slot) return;
    slot.h = clamp(mm / state.paper.heightMM, MIN_SIZE_FRACTION, 1 - slot.y);
    if (slotRatioLocked.value) {
      const wFrac = slotWidthFor(mm) / state.paper.widthMM;
      slot.w = clamp(wFrac, MIN_SIZE_FRACTION, 1 - slot.x);
    }
  },
});

/** Locks or unlocks the ratio of the selected slot at its current value. */
function onToggleSlotRatioLock(): void {
  toggleSlotRatioLock(wMm.value / Math.max(hMm.value, 1));
}

const customSlotRatioInput = ref('');

const slotRatioDisplay = computed(() =>
  selectedSlot.value && hMm.value > 0 ? (wMm.value / hMm.value).toFixed(2) : '—',
);

/**
 * Gives the selected slot a width to height ratio. It keeps the width and derives the height,
 * like the width field does while the ratio is locked.
 */
function applySlotRatio(ratio: number): void {
  const slot = selectedSlot.value;
  if (!slot) return;
  const hFrac = wMm.value / ratio / state.paper.heightMM;
  slot.h = clamp(hFrac, MIN_SIZE_FRACTION, 1 - slot.y);
}

/** Applies the typed ratio, if it can be read. */
function applyCustomSlotRatio(): void {
  const ratio = parseRatio(customSlotRatioInput.value);
  if (ratio === null) return;
  applySlotRatio(ratio);
}
</script>

<template>
  <EditorPanel
    :title="t('pages.layouts.panel.slot')"
    storage-key="layoutEditor.panel.slot"
    class="shrink-0"
  >
    <!-- Without this, removing the last slot would leave the panel without a way back besides the toolbar. -->
    <UEmpty
      v-if="activeSlots.length === 0"
      icon="i-lucide-square-dashed"
      :description="t('pages.layouts.noSlots')"
      :actions="[
        {
          label: t('pages.layouts.addSlot'),
          icon: 'i-lucide-square-plus',
          onClick: layout.addSlot,
        },
      ]"
      size="sm"
      variant="naked"
    />

    <EditorItemList
      v-else
      :items="items"
      :selected="selection.selected.value"
      sortable
      @select="onSelect"
      @toggle-lock="onToggleLock"
      @toggle-hidden="visibility.toggle"
      @move="layout.moveSlot"
    />

    <USeparator v-if="activeSlots.length" />

    <div v-if="selectedSlot" class="flex items-center justify-between">
      <h3 class="text-highlighted heading-card">
        {{ t('pages.layouts.slotLabel', { n: (selection.active.value ?? 0) + 1 }) }}
      </h3>
      <UTooltip
        :text="selectedSlot.locked ? t('pages.layouts.unlockSlot') : t('pages.layouts.lockSlot')"
        :kbds="keyLabels(shortcutKeys(LAYOUT_EDITOR_SHORTCUTS.lock))"
      >
        <UButton
          :icon="selectedSlot.locked ? 'i-lucide-lock' : 'i-lucide-lock-open'"
          color="neutral"
          :variant="selectedSlot.locked ? 'solid' : 'outline'"
          :aria-label="
            selectedSlot.locked ? t('pages.layouts.unlockSlot') : t('pages.layouts.lockSlot')
          "
          @click="toggleLock"
        />
      </UTooltip>
    </div>

    <template v-if="selection.selected.value.size > 1">
      <p class="text-muted text-sm">
        {{ t('pages.layouts.multipleSelected', { n: selection.selected.value.size }) }}
      </p>
      <EditorTransformFields
        v-if="movableBounds"
        :bounds="movableBounds"
        unit="mm"
        @change="layout.transformSelection"
      />
    </template>

    <template v-if="selectedSlot">
      <!-- A locked slot keeps its place, so its position, size, and ratio are hidden. -->
      <template v-if="!selectedSlot.locked">
        <div class="grid grid-cols-2 gap-4">
          <UFormField :label="t('pages.layouts.slot.xMm')">
            <UInputNumber v-model="xMm" orientation="vertical" :step="1" class="w-full" />
          </UFormField>
          <UFormField :label="t('pages.layouts.slot.yMm')">
            <UInputNumber v-model="yMm" orientation="vertical" :step="1" class="w-full" />
          </UFormField>
        </div>
        <div class="flex items-end gap-2">
          <UFormField :label="t('pages.layouts.slot.wMm')" class="min-w-0 flex-1">
            <UInputNumber v-model="wMm" orientation="vertical" :step="1" class="w-full" />
          </UFormField>
          <UTooltip
            :text="t(slotRatioLocked ? 'pages.layouts.unlockRatio' : 'pages.layouts.lockRatio')"
          >
            <UToggle
              :model-value="slotRatioLocked"
              :icon="slotRatioLocked ? 'i-lucide-link' : 'i-lucide-link-2-off'"
              color="neutral"
              variant="soft"
              active-color="primary"
              active-variant="soft"
              :aria-label="
                t(slotRatioLocked ? 'pages.layouts.unlockRatio' : 'pages.layouts.lockRatio')
              "
              @update:model-value="onToggleSlotRatioLock"
            />
          </UTooltip>
          <UFormField :label="t('pages.layouts.slot.hMm')" class="min-w-0 flex-1">
            <UInputNumber v-model="hMm" orientation="vertical" :step="1" class="w-full" />
          </UFormField>
        </div>

        <div class="flex flex-col gap-2">
          <p class="text-muted text-xs">
            {{ t('pages.layouts.slotRatioCurrent', { ratio: slotRatioDisplay }) }}
          </p>
          <div class="flex flex-wrap gap-2">
            <UButton
              v-for="preset in SLOT_RATIO_PRESETS"
              :key="preset.label"
              :label="preset.label"
              size="xs"
              color="neutral"
              variant="outline"
              @click="applySlotRatio(preset.ratio)"
            />
          </div>
          <UInput
            v-model="customSlotRatioInput"
            :placeholder="t('pages.layouts.slotRatioCustomPlaceholder')"
            size="sm"
            class="w-full"
            @keydown.enter.prevent="applyCustomSlotRatio"
            @blur="applyCustomSlotRatio"
          />
        </div>
      </template>

      <UButton
        :label="t('pages.layouts.removeSlot')"
        icon="i-lucide-trash-2"
        size="xs"
        color="error"
        variant="outline"
        class="w-fit"
        @click="layout.removeSelectedSlots"
      />
    </template>
  </EditorPanel>
</template>
