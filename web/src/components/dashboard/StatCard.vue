<script setup lang="ts">
import HelpHint from '@/components/form/HelpHint.vue';

defineProps<{
  icon: string;
  label: string;
  value: string;
  href?: string;
  help?: string;
}>();
</script>

<template>
  <UCard
    :as="href ? 'a' : 'div'"
    :href="href"
    :target="href ? '_blank' : undefined"
    :rel="href ? 'noopener noreferrer' : undefined"
    :class="href ? 'hover:ring-primary/50 transition-colors' : ''"
    :ui="{ body: 'flex items-center gap-4' }"
  >
    <div
      class="bg-primary/10 text-primary flex size-10 shrink-0 items-center justify-center rounded-lg"
    >
      <UIcon :name="icon" class="size-5" />
    </div>
    <div class="min-w-0">
      <div class="flex items-center gap-1.5">
        <p class="text-muted truncate text-sm">{{ label }}</p>
        <HelpHint v-if="help" :text="help" />
      </div>
      <p class="text-highlighted truncate text-lg font-semibold" :title="value">{{ value }}</p>
    </div>
  </UCard>
</template>
