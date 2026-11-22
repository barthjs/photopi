<script setup lang="ts">
import { useI18n } from 'vue-i18n';

// A bar at the bottom of the screen while photos are chosen, with their number, the actions on them,
// and a button that clears the choice.
defineProps<{
  count: number;
}>();

// Clear asks the parent to forget the choice
const emit = defineEmits<{
  clear: [];
}>();

const { t } = useI18n();
</script>

<template>
  <Transition
    enter-from-class="translate-y-4 opacity-0"
    leave-to-class="translate-y-4 opacity-0"
    enter-active-class="transition duration-150"
    leave-active-class="transition duration-150"
  >
    <div
      v-if="count > 0"
      class="bg-default ring-default fixed inset-x-4 bottom-4 z-50 mx-auto flex max-w-max flex-wrap items-center gap-2 rounded-lg p-2 shadow-lg ring"
      role="toolbar"
      :aria-label="t('gallery.selection.label')"
    >
      <UTooltip :text="t('gallery.selection.clear')">
        <UButton
          icon="i-lucide-x"
          color="neutral"
          variant="ghost"
          :aria-label="t('gallery.selection.clear')"
          @click="emit('clear')"
        />
      </UTooltip>
      <span class="text-highlighted me-2 text-sm font-medium tabular-nums">
        {{ t('gallery.selection.count', { count }, count) }}
      </span>
      <slot />
    </div>
  </Transition>
</template>
