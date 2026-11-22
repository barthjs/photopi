<script setup lang="ts">
// A square photo of a timeline that fills its tile, with a caption on a shaded strip at the bottom that shows on hover
// and badges in the top right corner. It leads to the page of the photo, or reports a click without one.
// A dimmed photo appears desaturated, such as one whose subject is gone.
defineProps<{
  // The photo, null without one
  src: string | null;
  alt: string;
  caption?: string;
  to?: string;
  dimmed?: boolean;
}>();

const emit = defineEmits<{
  open: [];
}>();

defineSlots<{
  badges?: () => unknown;
}>();
</script>

<template>
  <ULink
    :to="to"
    :as="to ? undefined : 'button'"
    class="group bg-elevated relative block h-full w-full"
    :aria-label="caption ?? alt"
    @click="!to && emit('open')"
  >
    <img
      v-if="src"
      :src="src"
      :alt="alt"
      loading="lazy"
      class="h-full w-full object-cover transition group-hover:brightness-95"
      :class="{ grayscale: dimmed }"
    />
    <div v-else class="text-dimmed flex h-full items-center justify-center">
      <UIcon name="i-lucide-image-off" class="size-6" />
    </div>
    <span
      v-if="caption"
      class="bg-inverted/60 text-inverted absolute inset-x-0 bottom-0 truncate px-2 py-1 text-left text-xs opacity-0 transition group-hover:opacity-100 group-focus-visible:opacity-100"
    >
      {{ caption }}
    </span>
    <span class="absolute top-1.5 right-1.5 flex gap-1">
      <slot name="badges" />
    </span>
  </ULink>
</template>
