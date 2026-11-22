<script setup lang="ts">
import { computed } from 'vue';

import EditorCanvasFrame from '@/components/editor/EditorCanvasFrame.vue';
import LayoutBrandingPreview from '@/components/layout-editor/LayoutBrandingPreview.vue';
import LayoutSlotBox from '@/components/layout-editor/LayoutSlotBox.vue';
import { EDITOR_SURFACE_ATTRIBUTE, useEditorCanvas } from '@/composables/editor/useEditorCanvas';
import type { EditorViewOptions } from '@/composables/editor/useEditorWorkspace';
import { useLayoutForm } from '@/composables/layout/useLayoutForm';
import { useSlotDrag } from '@/composables/layout/useSlotDrag';
import type { Bounds } from '@/lib/editor/geometry';
import { paperSize, slotBounds } from '@/lib/layout/layoutGeometry';

// The paper of the active page as the editing canvas, with zoom and pan
const props = defineProps<{
  view: EditorViewOptions;
  // Shows the page as printed, without the view options, and blocks editing
  viewMode: boolean;
}>();

// The zoom range relative to the paper fitted into the viewport
const MIN_ZOOM = 0.25;
const MAX_ZOOM = 6;

// The free space around the fitted paper, which keeps it clear of the rulers
const CANVAS_PADDING = 32;

// The distance of the grid lines on the paper
const GRID_STEP_MM = 10;

// The screen pixels per millimeter at 100 percent, the 96 pixels per inch of a screen
const PIXELS_PER_MM = 96 / 25.4;

const form = useLayoutForm();
const { state, editor, activeSlots } = form;
const { selection, visibility, sampleImage } = editor;

// The paper is fitted into the viewport in millimeters. The size of the viewport comes from the flex layout of the page,
// not from the position in the window, so scrolling does not resize the paper.
const canvas = useEditorCanvas({
  guides: editor.guides,
  contentWidth: () => state.paper.widthMM,
  contentHeight: () => state.paper.heightMM,
  padding: CANVAS_PADDING,
  actualScale: PIXELS_PER_MM,
  minZoom: MIN_ZOOM,
  maxZoom: MAX_ZOOM,
  selection,
  // Locked and hidden slots keep their place, so the marquee leaves them out
  items: () =>
    activeSlots.value.flatMap((slot, index): [number, Bounds][] =>
      slot.locked || visibility.isHidden(index) ? [] : [[index, slotBounds(slot, state.paper)]],
    ),
  gesture: () => editor.dragging.value,
  viewMode: () => props.viewMode,
});
const { viewport } = canvas;
const { zoom, pan, fitScale, spacePressed } = viewport;

const slotDrag = useSlotDrag(
  {
    snapping: () => props.view.snapping.value,
    toPaper: viewport.clientToContent,
    scale: viewport.scale,
    // The guides only snap while the rulers show them
    guides: () =>
      props.view.rulers.value
        ? editor.guides.targets()
        : {
            x: [],
            y: [],
          },
  },
  form,
);

// The slots that are drawn with their index on the page, without the hidden ones
const visibleSlots = computed(() =>
  activeSlots.value
    .map((slot, index) => ({
      slot,
      index,
    }))
    .filter(({ index }) => !visibility.isHidden(index)),
);

const gridShown = computed(() => props.view.grid.value && !props.viewMode);
const previewShown = computed(() => props.view.preview.value || props.viewMode);

// The paper is laid out at its fitted pixel size and only the zoom scales it,
// so text and borders inside keep their size on the screen at zoom 1.
const canvasDisplay = computed(() => ({
  width: state.paper.widthMM * fitScale.value,
  height: state.paper.heightMM * fitScale.value,
}));

// The grid lines keep one screen pixel at every zoom
const paperStyle = computed(() => ({
  width: `${canvasDisplay.value.width}px`,
  height: `${canvasDisplay.value.height}px`,
  transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom.value})`,
  transformOrigin: '0 0',
  '--grid-step': `${GRID_STEP_MM * fitScale.value}px`,
  '--grid-line': `${1 / zoom.value}px`,
}));

/** Fills the canvas with the selected slots. */
function zoomToSelection(): void {
  canvas.zoomToBounds(form.selectionBounds.value);
}

defineExpose({
  actualZoom: canvas.actualZoom,
  dragging: canvas.dragging,
  zoomBy: viewport.zoomBy,
  resetZoom: viewport.resetZoom,
  zoomToActual: canvas.zoomToActual,
  zoomToSelection,
});
</script>

<template>
  <!--
    From lg up, the canvas takes the space the flex layout gives it.
    On narrow screens it has a fixed height, so the cards below stay reachable.
  -->
  <EditorCanvasFrame
    :canvas="canvas"
    :content="paperSize(state.paper)"
    :feedback="slotDrag.feedback.value"
    unit="mm"
    :show-rulers="view.rulers.value"
    :show-guides="view.guides.value"
    :show-loupe="view.loupe.value && slotDrag.mode.value === 'resize'"
    :view-mode="viewMode"
    class="h-canvas min-h-72 lg:h-auto lg:min-h-0 lg:flex-1"
  >
    <div
      v-bind="{ [EDITOR_SURFACE_ATTRIBUTE]: '' }"
      class="bg-canvas-paper border-default dark:shadow-scrim/60 absolute top-0 left-0 rounded-sm border shadow-md"
      :class="{ 'layout-grid': gridShown }"
      :style="paperStyle"
    >
      <template v-if="previewShown">
        <LayoutSlotBox
          v-for="{ slot, index } in visibleSlots"
          :key="`preview-${index}`"
          :layout-slot="slot"
          :index="index"
          :selected="false"
          :show-handles="false"
          preview
          :outline-only="false"
          :image="sampleImage.url.value ?? null"
          :image-opacity="1"
        />
      </template>

      <template v-if="!viewMode">
        <LayoutSlotBox
          v-for="{ slot, index } in visibleSlots"
          :key="index"
          :layout-slot="slot"
          :index="index"
          :selected="selection.has(index)"
          :show-handles="selection.active.value === index && !slot.locked"
          :preview="false"
          :outline-only="view.preview.value"
          :image="sampleImage.url.value ?? null"
          :image-opacity="sampleImage.opacity.value"
          @box-down="slotDrag.startMove"
          @corner-down="slotDrag.startResize"
        />
      </template>

      <LayoutBrandingPreview
        :canvas-width="canvasDisplay.width"
        :canvas-height="canvasDisplay.height"
        :zoom="zoom"
        :pan-mode-active="spacePressed"
        :preview="viewMode"
      />
    </div>

    <template #loupe="{ strokeWidth }">
      <div class="bg-canvas-paper absolute inset-0"></div>
      <svg class="absolute inset-0 size-full overflow-visible">
        <template v-for="(slot, index) in activeSlots" :key="index">
          <image
            v-if="sampleImage.url.value"
            :href="sampleImage.url.value"
            :x="slot.x * state.paper.widthMM"
            :y="slot.y * state.paper.heightMM"
            :width="slot.w * state.paper.widthMM"
            :height="slot.h * state.paper.heightMM"
            preserveAspectRatio="xMidYMid slice"
            :opacity="sampleImage.opacity.value"
          />
          <rect
            :x="slot.x * state.paper.widthMM"
            :y="slot.y * state.paper.heightMM"
            :width="slot.w * state.paper.widthMM"
            :height="slot.h * state.paper.heightMM"
            class="loupe-outline"
            :stroke-width="strokeWidth"
          />
        </template>
      </svg>
    </template>
  </EditorCanvasFrame>
</template>

<style scoped>
/* The grid spans GRID_STEP_MM millimeters of the paper with lines of one screen pixel. */
.layout-grid {
  background-size: var(--grid-step) var(--grid-step);
  background-image:
    linear-gradient(
      to right,
      var(--color-canvas-grid) var(--grid-line),
      transparent var(--grid-line)
    ),
    linear-gradient(
      to bottom,
      var(--color-canvas-grid) var(--grid-line),
      transparent var(--grid-line)
    );
}

.loupe-outline {
  fill: none;
  stroke: var(--ui-primary);
}
</style>
