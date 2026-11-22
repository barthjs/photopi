<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { useLayoutForm } from '@/composables/layout/useLayoutForm';

// The pages of the layout. A run continues on the next page once the slots of one are filled.
const { t } = useI18n();
const { state, activePageIndex, setActivePage, addPage, removeActivePage } = useLayoutForm();

const pageOptions = computed(() =>
  state.pages.map((_, index) => ({
    label: t('pages.layouts.pageLabel', {
      n: index + 1,
    }),
    value: index,
  })),
);
</script>

<template>
  <div class="flex items-center gap-2">
    <UTabs
      :model-value="activePageIndex"
      :items="pageOptions"
      size="sm"
      class="min-w-0 flex-1"
      :ui="{ list: 'overflow-x-auto', trigger: 'shrink-0 grow-0' }"
      @update:model-value="setActivePage(Number($event))"
    />
    <UTooltip :text="t('pages.layouts.addPage')">
      <UButton
        icon="i-lucide-plus"
        size="sm"
        color="neutral"
        variant="outline"
        :aria-label="t('pages.layouts.addPage')"
        @click="addPage"
      />
    </UTooltip>
    <UTooltip :text="t('pages.layouts.removePage')">
      <UButton
        icon="i-lucide-trash-2"
        size="sm"
        color="error"
        variant="outline"
        :disabled="state.pages.length <= 1"
        :aria-label="t('pages.layouts.removePage')"
        @click="removeActivePage"
      />
    </UTooltip>
  </div>
</template>
