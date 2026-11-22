<script setup lang="ts" generic="T">
import { computed, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { useZoomPan } from '@/composables/ui/useZoomPan';

// A fullscreen viewer that swipes between the images, zooms into the current one with the wheel, a pinch,
// or a double click, and moves with the arrow keys. Only the current image and its neighbors load their full size,
// and near the last loaded image it asks for the next page. A bar below the image shows its position
// with the caption and the actions of the caller. The caller may place a badge beside the caption
// and wrap the image, such as to hide it until it is revealed. The details open in a panel beside the image,
// which moves below the bar on a phone, so the image shrinks and the actions stay in reach.
const props = defineProps<{
  items: T[];
  // The address of the full image
  src: (item: T) => string;
  alt: string;
  // Whether the parent has more images than it passed, which it loads on loadMore
  hasMore?: boolean;
}>();

// The shown image, -1 while the viewer is closed
const index = defineModel<number>('index', {
  required: true,
});

// LoadMore asks the parent for the next page once the viewer nears the last loaded image
const emit = defineEmits<{
  loadMore: [];
}>();

defineSlots<{
  // Wraps the image, whose element takes the attributes, such as its address and the transform of the zoom
  image?: (props: { item: T; attrs: ImageAttrs }) => unknown;
  badge?: (props: { item: T }) => unknown;
  caption?: (props: { item: T }) => unknown;
  actions?: (props: { item: T }) => unknown;
  // The details of the image in a panel beside it, which a button opens
  info?: (props: { item: T }) => unknown;
}>();

/** The attributes of the element of an image in the viewer. */
type ImageAttrs = {
  src: string;
  alt: string;
  draggable: 'false';
  class: (string | Record<string, boolean>)[];
  style?: Record<string, string>;
};

const { t } = useI18n();
const zoomPan = useZoomPan();
const carousel = useTemplateRef('carousel');

/** How close to the last loaded image the next page is asked for. */
const LOAD_MORE_DISTANCE = 2;

// The index the viewer opened with, so the carousel starts there without scrolling
// eslint-disable-next-line vue/no-ref-object-reactivity-loss -- the carousel starts at the index it opened with.
const startIndex = index.value;

const current = computed(() => props.items[index.value] ?? null);
const hasMultiple = computed(() => props.items.length > 1);
// The carousel holds the positions of the images, so it needs no shape of them
const slides = computed(() => props.items.map((_, position) => position));
// The carousel wraps around only once every image is loaded, so the end of a page leads to the next one
const loops = computed(() => hasMultiple.value && !props.hasMore);

// A deleted image leaves the list, so the viewer shows its neighbor or closes after the last one
watch(
  () => props.items.length,
  (length) => {
    if (index.value >= length) index.value = length - 1;
  },
);

watch(index, (value) => {
  zoomPan.reset();
  if (value >= 0 && carousel.value?.emblaApi?.selectedScrollSnap() !== value) {
    carousel.value?.emblaApi?.scrollTo(value);
  }
});

// The number of images the last request for the next page was made with, so each page is asked for once
let loadMoreRequestedAt = -1;

// Near the last loaded image the next page loads, so swiping on never reaches an end that is none
watch(
  [index, () => props.items.length, () => props.hasMore],
  ([value, length, hasMore]) => {
    if (!hasMore || value < length - LOAD_MORE_DISTANCE || loadMoreRequestedAt === length) return;
    loadMoreRequestedAt = length;
    emit('loadMore');
  },
  {
    immediate: true,
  },
);

/**
 * Reports whether a slide is the current image or one of its neighbors, which alone load their full size,
 * so a long gallery does not download every image at once.
 */
const isNear = (slideIndex: number) => {
  const distance = Math.abs(slideIndex - index.value);
  const wrapped = loops.value ? props.items.length - distance : distance;
  return Math.min(distance, wrapped) <= 1;
};

/** Returns the attributes of the image of a slide, the transform of the zoom on the current one. */
const imageAttrs = (item: T, slideIndex: number): ImageAttrs => ({
  src: props.src(item),
  alt: props.alt,
  draggable: 'false',
  class: [
    'max-h-full max-w-full rounded-lg object-contain select-none',
    {
      'transition-transform duration-150': !zoomPan.isActive.value,
    },
  ],
  style:
    slideIndex === index.value
      ? {
          transform: zoomPan.transform.value,
        }
      : undefined,
});

/** Follows the image the carousel swiped to. */
const onSelect = (selected: number) => {
  if (index.value >= 0) index.value = selected;
};

/**
 * Shows the next image and wraps around, so the gallery feels continuous in both directions.
 * The last loaded image stays while the next page loads.
 */
const next = () => {
  if (props.items.length === 0) return;
  if (!loops.value && index.value >= props.items.length - 1) return;
  index.value = (index.value + 1) % props.items.length;
};

/** Shows the previous image and wraps around at the start once every image is loaded. */
const previous = () => {
  if (props.items.length === 0) return;
  if (!loops.value && index.value <= 0) return;
  index.value = (index.value - 1 + props.items.length) % props.items.length;
};

/** Closes the viewer. */
const close = () => {
  index.value = -1;
};

// A zoomed image captures the pointer, which retargets the click to the slide, so only a press on the backdrop closes.
// The second click of a double click, which may have opened the viewer, never closes it.
let pressedBackdrop = false;

/** Notes whether the press started on the backdrop and passes it to the zoom of the current slide. */
const onSlidePointerDown = (event: PointerEvent, slideIndex: number) => {
  pressedBackdrop = event.target === event.currentTarget && !zoomPan.isZoomed.value;
  if (slideIndex === index.value) zoomPan.onPointerDown(event);
};

/** Closes the viewer on a click on the backdrop beside the image. */
const onSlideClick = (event: MouseEvent) => {
  if (pressedBackdrop && event.detail <= 1 && event.target === event.currentTarget) close();
  pressedBackdrop = false;
};

const root = useTemplateRef('root');

// The panel of the details, which stays open from one image to the next
const infoOpen = ref(false);

/**
 * Moves between the images with the arrow keys and closes with Escape, leaving the keys of a field or a dialog on top.
 */
const onKeydown = (event: KeyboardEvent) => {
  if (!(event.target instanceof Element) || isTyping(event.target)) return;
  const dialog = event.target.closest('[role="dialog"]');
  if (dialog && dialog !== root.value) return;

  if (event.key === 'Escape') close();
  // The focused carousel already moved and prevented the default of the key
  else if (event.defaultPrevented) return;
  else if (event.key === 'ArrowRight') next();
  else if (event.key === 'ArrowLeft') previous();
};

/** Reports whether an element takes typed text, so its keys never steer the viewer. */
const isTyping = (element: Element) =>
  element instanceof HTMLInputElement ||
  element instanceof HTMLTextAreaElement ||
  (element instanceof HTMLElement && element.isContentEditable);

// The viewer takes the focus, so the keys reach it also when it opens from inside another dialog
onMounted(() => {
  window.addEventListener('keydown', onKeydown);
  root.value?.focus();
});
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown));
</script>

<template>
  <Teleport to="body">
    <div
      ref="root"
      class="bg-scrim/90 z-viewer fixed inset-0 flex flex-col backdrop-blur-sm outline-none md:flex-row"
      role="dialog"
      aria-modal="true"
      tabindex="-1"
    >
      <!-- The image with its caption, beside the details, which take the full height -->
      <div class="flex min-h-0 min-w-0 flex-1 flex-col">
        <div class="relative min-h-0 flex-1">
          <!-- Swiping is off while the image is zoomed, so a pan does not change the image -->
          <UCarousel
            ref="carousel"
            v-slot="{ index: slideIndex }"
            :items="slides"
            :start-index="startIndex"
            :loop="loops"
            :watch-drag="hasMultiple && !zoomPan.isActive.value"
            :ui="{
              viewport: 'h-full',
              container: 'h-full ms-0',
              item: 'h-full ps-0',
            }"
            class="h-full"
            @select="onSelect"
          >
            <div
              class="flex h-full touch-none items-center justify-center overflow-hidden p-4 md:px-20"
              @click="onSlideClick"
              @pointerdown="onSlidePointerDown($event, slideIndex)"
              @pointermove="slideIndex === index && zoomPan.onPointerMove($event)"
              @pointerup="slideIndex === index && zoomPan.onPointerUp($event)"
              @pointercancel="slideIndex === index && zoomPan.onPointerUp($event)"
              @wheel="slideIndex === index && zoomPan.onWheel($event)"
            >
              <template v-if="isNear(slideIndex) && items[slideIndex]">
                <slot
                  name="image"
                  :item="items[slideIndex]"
                  :attrs="imageAttrs(items[slideIndex], slideIndex)"
                >
                  <img v-bind="imageAttrs(items[slideIndex], slideIndex)" />
                </slot>
              </template>
            </div>
          </UCarousel>

          <div class="absolute top-4 right-4 flex gap-1">
            <UToggle
              v-if="$slots.info"
              v-model="infoOpen"
              variant="ghost"
              color="neutral"
              active-variant="ghost"
              active-color="neutral"
              icon="i-lucide-info"
              size="lg"
              :aria-label="t('gallery.info')"
              class="text-on-scrim hover:bg-on-scrim/10"
            />
            <UButton
              variant="ghost"
              color="neutral"
              icon="i-lucide-x"
              size="lg"
              :aria-label="t('common.close')"
              class="text-on-scrim hover:bg-on-scrim/10"
              @click="close"
            />
          </div>

          <UButton
            v-if="hasMultiple"
            variant="ghost"
            color="neutral"
            icon="i-lucide-chevron-left"
            size="xl"
            :aria-label="t('gallery.previous')"
            class="text-on-scrim hover:bg-on-scrim/10 absolute top-1/2 left-2 -translate-y-1/2 md:left-6"
            @click="previous"
          />

          <UButton
            v-if="hasMultiple"
            variant="ghost"
            color="neutral"
            icon="i-lucide-chevron-right"
            size="xl"
            :aria-label="t('gallery.next')"
            class="text-on-scrim hover:bg-on-scrim/10 absolute top-1/2 right-2 -translate-y-1/2 md:right-6"
            @click="next"
          />
        </div>

        <div
          v-if="current && (hasMultiple || $slots.caption || $slots.badge || $slots.actions)"
          class="flex items-center gap-3 px-4 pt-2 pb-4"
        >
          <!-- The bar keeps one row, so the caption gives way and wraps its text before the actions do -->
          <div class="flex min-w-0 flex-1 items-center gap-3">
            <span v-if="hasMultiple" class="text-on-scrim shrink-0 text-sm tabular-nums">
              {{ index + 1 }} / {{ items.length }}
            </span>
            <slot name="caption" :item="current" />
            <slot name="badge" :item="current" />
          </div>
          <div class="flex shrink-0 items-center justify-end gap-1">
            <slot name="actions" :item="current" />
          </div>
        </div>
      </div>

      <!-- A panel beside the image on desktop and below the bar in the lower half on a phone -->
      <aside
        v-if="infoOpen && current && $slots.info"
        class="bg-default border-default flex max-h-1/2 shrink-0 flex-col rounded-t-lg md:max-h-none md:w-96 md:rounded-none md:border-s"
        :aria-label="t('gallery.info')"
      >
        <div class="border-default flex items-center gap-2 border-b px-4 py-3">
          <h2 class="text-highlighted min-w-0 flex-1 truncate font-semibold">
            {{ t('gallery.info') }}
          </h2>
          <UButton
            icon="i-lucide-x"
            color="neutral"
            variant="ghost"
            :aria-label="t('common.close')"
            @click="infoOpen = false"
          />
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto p-4">
          <slot name="info" :item="current" />
        </div>
      </aside>
    </div>
  </Teleport>
</template>
