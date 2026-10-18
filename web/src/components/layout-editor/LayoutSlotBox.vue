<script setup lang="ts">
import type { LayoutSlotRequest } from '@/api/types';
import type { SlotCorner } from '@/composables/useSlotDrag';

// A slot on the paper, positioned in fractions of the paper. Moving and resizing are left to the canvas,
// which receives the presses on the slot and on its corner handles.
defineProps<{
  layoutSlot: LayoutSlotRequest;
  index: number;
  // Highlights the border of every selected slot, also within a multi-selection.
  selected: boolean;
  // Shows the resize handles. Only a single selected, unlocked slot can be resized.
  showHandles: boolean;
  // Shows the slot as the print fills it, without its number and outline, and ignores the pointer.
  preview: boolean;
  // Draws only the outline, so the preview underneath stays visible.
  outlineOnly: boolean;
  // The sample photo, which fills the slot cropped from its center like the print, or null without one.
  image: string | null;
  imageOpacity: number;
}>();

const emit = defineEmits<{
  boxDown: [event: PointerEvent, index: number];
  cornerDown: [event: PointerEvent, index: number, corner: SlotCorner];
}>();

const corners: SlotCorner[] = ['nw', 'ne', 'sw', 'se'];

const cornerPositionClass: Record<SlotCorner, string> = {
  nw: 'left-0 top-0 -translate-x-1/2 -translate-y-1/2 cursor-nwse-resize',
  ne: 'right-0 top-0 translate-x-1/2 -translate-y-1/2 cursor-nesw-resize',
  sw: 'left-0 bottom-0 -translate-x-1/2 translate-y-1/2 cursor-nesw-resize',
  se: 'right-0 bottom-0 translate-x-1/2 translate-y-1/2 cursor-nwse-resize',
};
</script>

<template>
  <div
    v-if="preview"
    class="bg-canvas-photo text-canvas-photo-icon pointer-events-none absolute flex items-center justify-center overflow-hidden"
    :style="{
      left: `${layoutSlot.x * 100}%`,
      top: `${layoutSlot.y * 100}%`,
      width: `${layoutSlot.w * 100}%`,
      height: `${layoutSlot.h * 100}%`,
    }"
  >
    <img v-if="image" :src="image" alt="" class="size-full object-cover" />
    <UIcon v-else name="i-lucide-image" class="size-6" />
  </div>
  <!-- touch-none keeps a finger on a slot or a handle from scrolling the page instead of dragging. -->
  <div
    v-else
    class="absolute flex touch-none items-center justify-center overflow-hidden rounded-sm border-2 transition-colors select-none"
    :class="[
      outlineOnly ? 'bg-transparent' : 'bg-canvas-paper/70 dark:bg-canvas-paper/10',
      layoutSlot.locked ? 'cursor-not-allowed border-dashed' : 'cursor-grab active:cursor-grabbing',
      selected
        ? 'border-primary ring-primary/40 z-10 ring-2'
        : 'border-default hover:border-primary/50',
    ]"
    :style="{
      left: `${layoutSlot.x * 100}%`,
      top: `${layoutSlot.y * 100}%`,
      width: `${layoutSlot.w * 100}%`,
      height: `${layoutSlot.h * 100}%`,
    }"
    @pointerdown.left="emit('boxDown', $event, index)"
  >
    <img
      v-if="image && !outlineOnly"
      :src="image"
      alt=""
      class="pointer-events-none absolute inset-0 size-full object-cover"
      :style="{ opacity: imageOpacity }"
    />
    <div v-if="!outlineOnly" class="text-muted relative flex flex-col items-center gap-1">
      <UIcon :name="layoutSlot.locked ? 'i-lucide-lock' : 'i-lucide-image'" class="size-6" />
      <span class="text-xs font-medium">{{ index + 1 }}</span>
    </div>

    <template v-if="showHandles">
      <div
        v-for="corner in corners"
        :key="corner"
        class="bg-primary ring-canvas-handle-ring absolute size-3 touch-none rounded-full ring-2"
        :class="cornerPositionClass[corner]"
        @pointerdown.left.stop="emit('cornerDown', $event, index, corner)"
      />
    </template>
  </div>
</template>
