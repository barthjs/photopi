<script setup lang="ts">
import { watch } from 'vue';
import { useI18n } from 'vue-i18n';

import EditorPanel from '@/components/editor/EditorPanel.vue';
import { useAspectRatioLock } from '@/composables/useAspectRatioLock';
import {
  LAYOUT_LIMITS,
  PAPER_MAX_MM,
  PAPER_MIN_MM,
  useLayoutForm,
} from '@/composables/useLayoutForm';
import { roundMm } from '@/lib/layoutGeometry';
import { countLabel } from '@/lib/form/validation';

// Common paper sizes of photo printers, to set the paper directly.
const PAPER_PRESETS = [
  {
    label: '10 × 15 cm',
    widthMM: 100,
    heightMM: 150,
  },
  {
    label: '5 × 15 cm',
    widthMM: 50,
    heightMM: 150,
  },
  {
    label: '13 × 18 cm',
    widthMM: 130,
    heightMM: 180,
  },
  {
    label: '15 × 15 cm',
    widthMM: 150,
    heightMM: 150,
  },
  {
    label: 'A5',
    widthMM: 148,
    heightMM: 210,
  },
  {
    label: 'A4',
    widthMM: 210,
    heightMM: 297,
  },
];

// The identity of the layout and its paper.
const { t } = useI18n();
const { state, fieldProps, isCreating, idManuallyEdited, openCount } = useLayoutForm();

// While the ratio is locked, the other dimension follows. Both fields stay in range,
// also with the value the lock derives.
const {
  locked: paperRatioLocked,
  lock: lockPaperRatio,
  toggle: togglePaperRatioLock,
} = useAspectRatioLock(
  {
    get: () => state.paper.widthMM,
    set: (mm) => {
      state.paper.widthMM = clampPaperDimension(mm);
    },
  },
  {
    get: () => state.paper.heightMM,
    set: (mm) => {
      state.paper.heightMM = clampPaperDimension(mm);
    },
  },
);

// An emptied field becomes undefined, which falls back to the minimum
watch(
  () => state.paper.widthMM,
  (width) => {
    if (!Number.isFinite(width)) state.paper.widthMM = PAPER_MIN_MM;
  },
);
watch(
  () => state.paper.heightMM,
  (height) => {
    if (!Number.isFinite(height)) state.paper.heightMM = PAPER_MIN_MM;
  },
);

// Every opened layout starts with the ratio of its paper locked, since a paper keeps its shape more often than not
// The watcher runs at once, before the fields derive one dimension from the ratio of the previous layout.
watch(openCount, () => lockPaperRatio(), {
  immediate: true,
  flush: 'sync',
});

/**
 * applyPreset sets the paper to a common size and locks the ratio at it. Locking before the watchers of the fields
 * run keeps them from deriving one dimension from the previous ratio.
 */
function applyPreset(preset: { widthMM: number; heightMM: number }): void {
  state.paper.widthMM = preset.widthMM;
  state.paper.heightMM = preset.heightMM;
  lockPaperRatio();
}

/** isCurrentPreset reports whether the paper has the size of a common one. */
function isCurrentPreset(preset: { widthMM: number; heightMM: number }): boolean {
  return state.paper.widthMM === preset.widthMM && state.paper.heightMM === preset.heightMM;
}

/**
 * clampPaperDimension keeps a paper dimension in range. The input only clamps a parsed value,
 * while an emptied field becomes undefined, so a value that is not a number falls back to the minimum.
 */
function clampPaperDimension(mm: number): number {
  if (!Number.isFinite(mm)) return PAPER_MIN_MM;
  return Math.min(Math.max(roundMm(mm), PAPER_MIN_MM), PAPER_MAX_MM);
}
</script>

<template>
  <EditorPanel
    :title="t('pages.layouts.panel.layout')"
    storage-key="layoutEditor.panel.layout"
    class="shrink-0"
  >
    <UFormField
      :label="t('fields.name')"
      :hint="countLabel(state.name, LAYOUT_LIMITS.name)"
      name="name"
      required
    >
      <UInput
        v-model="state.name"
        v-bind="fieldProps('name')"
        :maxlength="LAYOUT_LIMITS.name"
        class="w-full"
      />
    </UFormField>
    <UFormField :label="t('fields.id')" name="id" required>
      <UInput
        v-model="state.id"
        :disabled="!isCreating"
        :placeholder="t('pages.layouts.idPlaceholder')"
        class="w-full"
        @input="idManuallyEdited = true"
      />
    </UFormField>

    <div class="flex flex-col gap-2">
      <div class="flex items-end gap-2">
        <UFormField :label="t('fields.paper.widthMM')" name="paper.widthMM" class="min-w-0 flex-1">
          <UInputNumber
            orientation="vertical"
            v-model="state.paper.widthMM"
            v-bind="fieldProps('paper.widthMM')"
            :min="PAPER_MIN_MM"
            :max="PAPER_MAX_MM"
            class="w-full"
          />
        </UFormField>
        <UTooltip
          :text="t(paperRatioLocked ? 'pages.layouts.unlockRatio' : 'pages.layouts.lockRatio')"
        >
          <UButton
            :icon="paperRatioLocked ? 'i-lucide-link' : 'i-lucide-link-2-off'"
            :color="paperRatioLocked ? 'primary' : 'neutral'"
            variant="soft"
            :aria-label="
              t(paperRatioLocked ? 'pages.layouts.unlockRatio' : 'pages.layouts.lockRatio')
            "
            :aria-pressed="paperRatioLocked"
            @click="togglePaperRatioLock"
          />
        </UTooltip>
        <UFormField
          :label="t('fields.paper.heightMM')"
          name="paper.heightMM"
          class="min-w-0 flex-1"
        >
          <UInputNumber
            orientation="vertical"
            v-model="state.paper.heightMM"
            v-bind="fieldProps('paper.heightMM')"
            :min="PAPER_MIN_MM"
            :max="PAPER_MAX_MM"
            class="w-full"
          />
        </UFormField>
      </div>
      <div class="grid grid-cols-2 gap-1 @xs:grid-cols-3">
        <UButton
          v-for="preset in PAPER_PRESETS"
          :key="preset.label"
          :label="preset.label"
          size="xs"
          class="justify-center px-1.5"
          :color="isCurrentPreset(preset) ? 'primary' : 'neutral'"
          :variant="isCurrentPreset(preset) ? 'soft' : 'outline'"
          @click="applyPreset(preset)"
        />
      </div>
    </div>
  </EditorPanel>
</template>
