<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import EditorHelpFrame, { type HelpShortcut } from '@/components/editor/EditorHelpFrame.vue';
import { shortcutKeys } from '@/lib/editor/shortcuts';
import { LAYOUT_EDITOR_SHORTCUTS } from '@/lib/layout/layoutEditorShortcuts';
import { objectKeys } from '@/lib/types/guards';

// Explains what the layout editor is for and the usual workflow, and lists its keyboard shortcuts
const open = defineModel<boolean>('open', {
  required: true,
});

const { t } = useI18n();

// The number of steps of the guide in the locales
const STEP_COUNT = 6;

const steps = computed(() =>
  Array.from(
    {
      length: STEP_COUNT,
    },
    (_, index) => t(`pages.layouts.guide.steps.${index + 1}`),
  ),
);

// The shortcuts of the layout editor itself and how the slots are selected and moved
const shortcuts = computed<HelpShortcut[]>(() => [
  ...objectKeys(LAYOUT_EDITOR_SHORTCUTS).map((shortcut) => ({
    keys: shortcutKeys(LAYOUT_EDITOR_SHORTCUTS[shortcut]),
    label: t(`pages.layouts.guide.shortcuts.${shortcut}`),
  })),
  {
    keys: ['1', '…', '9'],
    label: t('pages.layouts.guide.shortcuts.selectSlot'),
  },
  {
    keys: ['arrows'],
    label: t('pages.layouts.guide.shortcuts.nudge'),
  },
  {
    keys: ['shift', 'arrows'],
    label: t('pages.layouts.guide.shortcuts.nudgeLarge'),
  },
  {
    keys: ['shift'],
    label: t('pages.layouts.guide.shortcuts.shiftSelect'),
  },
  {
    keys: ['shift'],
    label: t('pages.layouts.guide.shortcuts.shiftResize'),
  },
  {
    keys: ['alt'],
    label: t('pages.layouts.guide.shortcuts.alt'),
  },
]);
</script>

<template>
  <EditorHelpFrame v-model:open="open" :shortcuts="shortcuts">
    <template #guide>
      <p>{{ t('pages.layouts.guide.intro') }}</p>

      <ol class="flex list-decimal flex-col gap-2 pl-5">
        <li v-for="(step, index) in steps" :key="index">{{ step }}</li>
      </ol>

      <div class="flex flex-col gap-2">
        <h3 class="text-highlighted heading-card">
          {{ t('pages.layouts.guide.layoutsTitle') }}
        </h3>
        <p>{{ t('pages.layouts.guide.layouts') }}</p>
        <p>{{ t('pages.layouts.guide.importExport') }}</p>
      </div>

      <div class="flex flex-col gap-2">
        <h3 class="text-highlighted heading-card">
          {{ t('pages.layouts.guide.viewTitle') }}
        </h3>
        <p class="whitespace-pre-line">{{ t('pages.layouts.guide.view') }}</p>
      </div>
    </template>
  </EditorHelpFrame>
</template>
