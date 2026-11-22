<script setup lang="ts">
import { watch } from 'vue';
import { useI18n } from 'vue-i18n';

import EditorPanel from '@/components/editor/EditorPanel.vue';
import { useAspectRatioLock } from '@/composables/layout/useAspectRatioLock';
import {
  LAYOUT_LIMITS,
  PAPER_MAX_MM,
  PAPER_MIN_MM,
  useLayoutForm,
} from '@/composables/layout/useLayoutForm';
import { roundMm } from '@/lib/layout/layoutGeometry';
import { countLabel } from '@/lib/form/validation';

// Common paper sizes of photo printers, to set the paper directly
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

// The identity of the layout and its paper
const { t } = useI18n();
const { state, fieldProps, openCount, isDraft, isDirty } = useLayoutForm();

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

// Every opened layout starts with the ratio of its paper locked, since a paper keeps its shape more often than not.
// The watcher runs at once, before the fields derive one dimension from the ratio of the previous layout.
watch(openCount, () => lockPaperRatio(), {
  immediate: true,
  flush: 'sync',
});

/**
 * Sets the paper to a common size and locks the ratio at it before the watchers of the fields run,
 * so they do not derive one dimension from the previous ratio.
 */
function applyPreset(preset: { widthMM: number; heightMM: number }): void {
  state.paper.widthMM = preset.widthMM;
  state.paper.heightMM = preset.heightMM;
  lockPaperRatio();
}

/** Reports whether the paper has the size of a common one. */
function isCurrentPreset(preset: { widthMM: number; heightMM: number }): boolean {
  return state.paper.widthMM === preset.widthMM && state.paper.heightMM === preset.heightMM;
}

/**
 * Keeps a paper dimension in range. The input only clamps a parsed value,
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
    <!-- The name is a field below, so the header only marks an unsaved draft and unsaved changes. -->
    <template #summary>
      <UBadge v-if="isDraft" :label="t('pages.layouts.draft')" color="info" variant="solid" />
      <UBadge v-if="isDirty" :label="t('pages.layouts.changed')" color="warning" variant="solid" />
    </template>

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
    <UFormField
      :label="t('fields.description')"
      :hint="countLabel(state.description ?? '', LAYOUT_LIMITS.description)"
      name="description"
    >
      <UInput
        :model-value="state.description ?? ''"
        v-bind="fieldProps('description')"
        :maxlength="LAYOUT_LIMITS.description"
        class="w-full"
        @update:model-value="(value: string) => (state.description = value || null)"
      />
    </UFormField>

    <div class="flex flex-col gap-2">
      <div class="flex items-end gap-2">
        <UFormField :label="t('fields.paper.widthMM')" name="paper.widthMM" class="min-w-0 flex-1">
          <UInputNumber
            v-model="state.paper.widthMM"
            orientation="vertical"
            v-bind="fieldProps('paper.widthMM')"
            :min="PAPER_MIN_MM"
            :max="PAPER_MAX_MM"
            class="w-full"
          />
        </UFormField>
        <UTooltip
          :text="t(paperRatioLocked ? 'pages.layouts.unlockRatio' : 'pages.layouts.lockRatio')"
        >
          <UToggle
            :model-value="paperRatioLocked"
            :icon="paperRatioLocked ? 'i-lucide-link' : 'i-lucide-link-2-off'"
            color="neutral"
            variant="soft"
            active-color="primary"
            active-variant="soft"
            :aria-label="
              t(paperRatioLocked ? 'pages.layouts.unlockRatio' : 'pages.layouts.lockRatio')
            "
            @update:model-value="togglePaperRatioLock"
          />
        </UTooltip>
        <UFormField
          :label="t('fields.paper.heightMM')"
          name="paper.heightMM"
          class="min-w-0 flex-1"
        >
          <UInputNumber
            v-model="state.paper.heightMM"
            orientation="vertical"
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
