<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { useLayoutForm } from '@/composables/layout/useLayoutForm';
import { usePointerDrag } from '@/composables/editor/usePointerDrag';
import { clamp } from '@/lib/editor/geometry';

const props = defineProps<{
  // The size of the paper on screen before zooming, in pixels
  canvasWidth: number;
  canvasHeight: number;
  // The zoom of the canvas. Pointer deltas arrive in screen pixels and are divided by it.
  zoom: number;
  // While the canvas is panned, a pointer press must not move the caption
  panModeActive: boolean;
  // Shows the caption as printed, without its background, and ignores the pointer
  preview: boolean;
}>();

// A point is 0.3528 mm
const MM_PER_PT = 0.3528;

// The caption on the canvas, placed and drawn like the server prints it. It sits inside the paper,
// so it zooms and pans with it. Dragging it moves the caption.
const { t } = useI18n();
const { state, editor, brandingFontSizePt, brandingColor } = useLayoutForm();
const { dragging } = editor;

// The editor does not know the event name of the configuration, so the preview uses a placeholder
const previewText = computed(() =>
  state.branding.template
    .replaceAll('{{eventName}}', t('pages.layouts.brandingPreviewEventName'))
    .replaceAll('{{date}}', new Date().toLocaleDateString()),
);

// Like the print, the caption is centered on its point and rotated around it.
// The font size follows the physical size of the paper through the current pixels per millimeter.
const previewStyle = computed(() => {
  const pxPerMm = props.canvasHeight / Math.max(state.paper.heightMM, 1);
  const fontSizePx = Math.max(8, brandingFontSizePt.value * MM_PER_PT * pxPerMm);
  return {
    left: `${state.branding.x * 100}%`,
    top: `${state.branding.y * 100}%`,
    transform: `translate(-50%, -50%) rotate(${state.branding.rotationDeg || 0}deg)`,
    fontSize: `${fontSizePx}px`,
    color: brandingColor.value,
  };
});

// The start position is kept, so every move is applied to it instead of adding up rounding errors
let dragStart = {
  x: 0.5,
  y: 0.5,
};
const drag = usePointerDrag({
  onStart: () => {
    dragStart = {
      x: state.branding.x,
      y: state.branding.y,
    };
    dragging.value = true;
  },
  onMove: (deltaX, deltaY) => {
    const dxFrac = deltaX / props.zoom / props.canvasWidth;
    const dyFrac = deltaY / props.zoom / props.canvasHeight;
    state.branding.x = clamp(dragStart.x + dxFrac, 0, 1);
    state.branding.y = clamp(dragStart.y + dyFrac, 0, 1);
  },
  onEnd: () => {
    dragging.value = false;
    editor.history.commit();
  },
});

/** Starts moving the caption with the primary button. */
function onPointerDown(event: PointerEvent): void {
  if (event.button !== 0 || props.panModeActive) return;
  drag.start(event);
}
</script>

<template>
  <div
    v-if="state.branding.enabled && previewText"
    class="absolute z-20 rounded-xs px-1.5 py-0.5 whitespace-nowrap"
    :class="preview ? 'pointer-events-none' : 'bg-canvas-paper/75 cursor-move'"
    :style="previewStyle"
    @pointerdown="onPointerDown"
  >
    {{ previewText }}
  </div>
</template>
