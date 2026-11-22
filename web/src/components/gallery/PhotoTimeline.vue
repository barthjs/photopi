<script setup lang="ts" generic="T">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useElementSize } from '@vueuse/core';

import {
  inPeriod,
  parentPeriod,
  periodOffset,
  periodStart,
  timelineColumns,
  timelineRows,
  TIMELINE_ZOOMS,
  type TimelinePeriod,
  type TimelineRow,
  type TimelineZoom,
} from '@/lib/gallery/timeline';
import { selectRange, toggleKey } from '@/lib/gallery/selection';
import { isOneOf, isRecord } from '@/lib/types/guards';

// The photos of a list as a timeline in the manner of a photo app: a heading per period of the zoom level,
// dense rows of square tiles below, and a scale of the periods on the right that jumps through the list.
// It only renders the rows in view and asks for the next page once the end comes into view.
// The caller renders a tile in the slot. A selectable timeline chooses photos like a photo app:
// the checkbox of a tile shows on hover, a long press chooses on a touch screen, a click with the shift key chooses a range,
// and while photos are chosen a click on a tile chooses it instead of opening it.
// The arrow keys move between the tiles, the control key with the wheel or a pinch zooms,
// and the scale on the right, which replaces the scrollbar, can be dragged.
// With a key it restores the loaded photos and the scroll position when the page is shown again.
const props = defineProps<{
  items: T[];
  itemKey: (item: T) => string;
  itemDate: (item: T) => Date;
  // Every period of the scale with its number of photos, in the order of the list
  periods: TimelinePeriod[];
  // The zoom level of the periods of the scale, month by default
  scale?: TimelineZoom;
  // The zoom levels to choose from, from the smallest tiles to the largest, year, month, and day by default
  zooms?: TimelineZoom[];
  hasMore: boolean;
  loading: boolean;
  // Loads the list until it holds the number of photos, so a jump can reach a period further down
  ensureLoaded: (count: number) => Promise<void>;
  // Lets the photos be chosen for an action on several of them
  selectable?: boolean;
  // Names the timeline in the session storage, so it returns to the same place, such as the address of the page
  restoreKey?: string;
  // Fills the height of its parent, such as a side panel, instead of the height of a panel on a page
  fill?: boolean;
}>();

const zoom = defineModel<TimelineZoom>('zoom', {
  default: 'month',
});

const selected = defineModel<string[]>('selected', {
  default: () => [],
});

const emit = defineEmits<{
  loadMore: [];
}>();

defineSlots<{
  tile: (props: { item: T }) => unknown;
}>();

const { t, locale } = useI18n();

const zoomLevels = computed<TimelineZoom[]>(() => props.zooms ?? ['year', 'month', 'day']);
const zoomItems = computed(() =>
  zoomLevels.value.map((level) => ({
    label: t(`gallery.zoom.${level}`),
    value: level,
  })),
);

/** Switches to the chosen zoom level. A press on the active one keeps it. */
function onZoomSelect(value: unknown): void {
  if (isOneOf(zoomLevels.value, value)) zoom.value = value;
}

// The keys of the photos in their order, and the position of each, so a range and the keyboard find their tiles
const keys = computed(() => props.items.map(props.itemKey));
const positions = computed(() => new Map(keys.value.map((key, position) => [key, position])));

/** The gap between two tiles in pixels. */
const TIMELINE_GAP = 4;

/** The estimated height of a heading in pixels. */
const HEADER_HEIGHT = 48;

/** The number of rows before the end at which the next page loads. */
const LOAD_AHEAD = 8;

/** How long a touch rests on a tile before it chooses the photo. */
const LONG_PRESS_MS = 500;

/** The change of the distance of two fingers that zooms one level. */
const PINCH_STEP = 1.4;

const container = useTemplateRef<HTMLElement>('container');
const { width } = useElementSize(container);

const columns = computed(() => timelineColumns(width.value, zoom.value));
const tileSize = computed(
  () => (width.value - TIMELINE_GAP * (columns.value - 1)) / columns.value || 0,
);
const rows = computed(() => timelineRows(props.items, props.itemDate, zoom.value, columns.value));

const scrollArea = useTemplateRef<{
  $el?: HTMLElement;
  virtualizer?: {
    getVirtualItems: () => { index: number; end: number }[];
    scrollOffset: number | null;
    getTotalSize: () => number;
    scrollToIndex: (index: number, options?: { align: 'start' | 'auto' }) => void;
  };
}>('scrollArea');

// A timeline on a page is as tall as its rows up to the height of a panel, so a few photos leave no gap above the footer.
// The height follows the virtualizer after each rendering, since it measures the rows then.
const totalHeight = ref(0);
watch(
  () => scrollArea.value?.virtualizer?.getTotalSize() ?? 0,
  (size) => {
    totalHeight.value = size;
  },
  {
    flush: 'post',
  },
);
const pageHeight = computed(() =>
  props.fill
    ? undefined
    : {
        height: `${totalHeight.value}px`,
        maxHeight: 'var(--height-panel)',
      },
);

// The rows the virtualizer renders, a few beyond the view on each side, which decide the loading of the next page
const visible = computed(() => scrollArea.value?.virtualizer?.getVirtualItems() ?? []);

// The first row in view, which decides the period on top and the position on the scale
const firstVisible = computed(() => {
  const offset = scrollArea.value?.virtualizer?.scrollOffset ?? 0;
  return visible.value.find((item) => item.end > offset)?.index ?? 0;
});

watch(
  () => visible.value.at(-1)?.index ?? 0,
  (last) => {
    if (props.hasMore && !props.loading && last >= rows.value.length - LOAD_AHEAD) emit('loadMore');
  },
);

// The heading of the period at the top of the view
const currentHeader = computed(() => {
  for (let index = Math.min(firstVisible.value, rows.value.length - 1); index >= 0; index -= 1) {
    const row = rows.value[index];
    if (row?.kind === 'header') return row;
  }
  return null;
});

// The heading above the list repeats the period on top only once its own heading left the view
const pinnedHeader = computed(() =>
  rows.value[firstVisible.value]?.kind === 'header' ? null : currentHeader.value,
);

/** The formats of the name of a period per zoom level. */
const PERIOD_FORMATS: Record<TimelineZoom, Intl.DateTimeFormatOptions> = {
  year: {
    year: 'numeric',
  },
  month: {
    year: 'numeric',
    month: 'long',
  },
  week: {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  },
  day: {
    weekday: 'short',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  },
};

/** Names a period of the zoom level in the locale, a week by its first and its last day. */
function periodLabel(start: Date, level: TimelineZoom): string {
  const format = new Intl.DateTimeFormat(locale.value, PERIOD_FORMATS[level]);
  if (level !== 'week') return format.format(start);
  const end = new Date(start);
  end.setDate(end.getDate() + 6);
  return format.formatRange(start, end);
}

/** Names the period of a heading in the locale. */
function headerLabel(row: Extract<TimelineRow<T>, { kind: 'header' }>): string {
  return periodLabel(periodStart(row.period), zoom.value);
}

/**
 * Returns the number of photos of the period of a heading. A period as large as those of the scale
 * or larger sums their counts, so it is complete before its photos load, and a smaller one counts the loaded photos.
 */
function headerCount(row: Extract<TimelineRow<T>, { kind: 'header' }>): number {
  if (TIMELINE_ZOOMS.indexOf(zoom.value) > TIMELINE_ZOOMS.indexOf(props.scale ?? 'month')) {
    return row.items.length;
  }
  return props.periods
    .filter((entry) => inPeriod(periodStart(entry.period), row.period, zoom.value))
    .reduce((sum, entry) => sum + entry.count, 0);
}

/** The short formats of the larger periods that the scale names, per zoom level. */
const SCALE_FORMATS: Record<TimelineZoom, Intl.DateTimeFormatOptions> = {
  year: {
    year: 'numeric',
  },
  month: {
    month: 'short',
  },
  week: {
    day: 'numeric',
    month: 'short',
  },
  day: {
    day: 'numeric',
    month: 'short',
  },
};

// The periods as marks of the scale, placed by the share of the photos before them,
// with the short name of the larger period at its first mark, such as the year at its first month.
const marks = computed(() => {
  const level = props.scale ?? 'month';
  const parentLevel = TIMELINE_ZOOMS[Math.max(TIMELINE_ZOOMS.indexOf(level) - 1, 0)] ?? 'year';
  const total = props.periods.reduce((sum, entry) => sum + entry.count, 0) || 1;
  let before = 0;
  let previousParent = '';
  return props.periods.map((entry) => {
    const parent = parentPeriod(entry.period);
    const mark = {
      period: entry.period,
      top: (before / total) * 100,
      parent:
        parent === previousParent
          ? null
          : new Intl.DateTimeFormat(locale.value, SCALE_FORMATS[parentLevel]).format(
              periodStart(parent),
            ),
      label: periodLabel(periodStart(entry.period), level),
    };
    previousParent = parent;
    before += entry.count;
    return mark;
  });
});

/** Loads the list down to the period of the scale and scrolls to the first heading that holds it or lies in it. */
async function jumpTo(period: string): Promise<void> {
  await props.ensureLoaded(periodOffset(props.periods, period) + 1);
  const level = props.scale ?? 'month';
  const index = rows.value.findIndex(
    (row) =>
      row.kind === 'header' &&
      (inPeriod(periodStart(period), row.period, zoom.value) || inPeriod(row.start, period, level)),
  );
  if (index >= 0) {
    scrollArea.value?.virtualizer?.scrollToIndex(index, {
      align: 'start',
    });
  }
}

/** Moves the zoom one level towards larger tiles, or towards smaller ones with a negative step. */
function zoomBy(step: number): void {
  const levels = zoomLevels.value;
  const index = levels.indexOf(zoom.value) + step;
  const next = levels[Math.min(Math.max(index, 0), levels.length - 1)];
  if (next) zoom.value = next;
}

/** Zooms with the wheel while the control key is held, as a photo app does. */
function onWheel(event: WheelEvent): void {
  if (!event.ctrlKey) return;
  event.preventDefault();
  zoomBy(event.deltaY < 0 ? 1 : -1);
}

const selectedKeys = computed(() => new Set(selected.value));
const isSelecting = computed(() => selected.value.length > 0);

// The photo the last choice started from, which a click with the shift key extends to a range
const anchor = ref<string | null>(null);

/** Chooses a photo or leaves it out, or chooses the range from the anchor with the shift key. */
function choose(item: T, range: boolean): void {
  const key = props.itemKey(item);
  selected.value = range
    ? selectRange(selected.value, keys.value, anchor.value, key)
    : toggleKey(selected.value, key);
  anchor.value = key;
}

/** Chooses the photo while photos are chosen, so the click does not open it. */
function onTileClick(event: MouseEvent, item: T): void {
  if (!props.selectable || (!isSelecting.value && !event.shiftKey)) return;
  event.preventDefault();
  event.stopPropagation();
  choose(item, event.shiftKey);
}

// A long press on a touch screen chooses a photo, as the checkbox only shows on hover
let pressTimer: ReturnType<typeof setTimeout> | undefined;
let pressed = false;

/** Starts the long press of a touch on a tile. */
function onTilePointerDown(event: PointerEvent, item: T): void {
  if (!props.selectable || event.pointerType !== 'touch') return;
  pressed = false;
  pressTimer = setTimeout(() => {
    pressed = true;
    choose(item, false);
  }, LONG_PRESS_MS);
}

/** Ends a long press, whose click must not open the photo afterwards. */
function onTilePointerEnd(event: PointerEvent): void {
  clearTimeout(pressTimer);
  if (pressed) {
    event.preventDefault();
    pressed = false;
  }
}

/** Reports whether every loaded photo of a period is chosen. */
function isGroupSelected(items: T[]): boolean {
  return items.every((item) => selectedKeys.value.has(props.itemKey(item)));
}

/** Chooses every loaded photo of a period, or leaves them all out once all are chosen. */
function toggleGroup(items: T[]): void {
  const groupKeys = items.map(props.itemKey);
  if (isGroupSelected(items)) {
    selected.value = selected.value.filter((key) => !groupKeys.includes(key));
    return;
  }
  selected.value = [...new Set([...selected.value, ...groupKeys])];
}

/** Returns the row that holds the photo at the position. */
function rowOf(position: number): number {
  const key = keys.value[position];
  return rows.value.findIndex(
    (row) => row.kind === 'row' && row.items.some((item) => props.itemKey(item) === key),
  );
}

/**
 * Scrolls to the photo at the position and focuses the link or the button of its tile.
 * The tile only exists once its row is drawn, so the focus waits for the next rendering.
 */
async function focusTile(position: number): Promise<void> {
  const row = rowOf(position);
  if (row < 0) return;
  scrollArea.value?.virtualizer?.scrollToIndex(row, {
    align: 'auto',
  });
  await nextTick();
  const tile = container.value?.querySelector<HTMLElement>(`[data-tile="${position}"]`);
  tile?.querySelector<HTMLElement>('a, button')?.focus();
}

/** Moves between the tiles with the arrow keys, chooses with space, and clears the choice with escape. */
function onKeydown(event: KeyboardEvent): void {
  const tile =
    event.target instanceof Element ? event.target.closest<HTMLElement>('[data-tile]') : null;
  if (event.key === 'Escape' && isSelecting.value) {
    selected.value = [];
    return;
  }
  if (!tile) return;
  const position = Number(tile.dataset.tile);
  const steps: Record<string, number> = {
    ArrowLeft: -1,
    ArrowRight: 1,
    ArrowUp: -columns.value,
    ArrowDown: columns.value,
  };
  const step = steps[event.key];
  if (step !== undefined) {
    event.preventDefault();
    const target = Math.min(Math.max(position + step, 0), props.items.length - 1);
    if (target >= props.items.length - columns.value && props.hasMore) emit('loadMore');
    void focusTile(target);
    return;
  }
  const item = props.items[position];
  if (event.key === ' ' && props.selectable && item !== undefined) {
    event.preventDefault();
    choose(item, event.shiftKey);
  }
}

// Two fingers that move apart or together zoom the timeline, one level per step of the distance
let pinchDistance = 0;

/** Returns the distance of the first two touches. */
function touchDistance(event: TouchEvent): number {
  const [first, second] = [event.touches[0], event.touches[1]];
  if (!first || !second) return 0;
  return Math.hypot(first.clientX - second.clientX, first.clientY - second.clientY);
}

/** Notes the distance of a pinch. */
function onTouchStart(event: TouchEvent): void {
  pinchDistance = event.touches.length === 2 ? touchDistance(event) : 0;
}

/** Zooms once a pinch moved far enough and starts the next step from there. */
function onTouchMove(event: TouchEvent): void {
  if (pinchDistance === 0 || event.touches.length !== 2) return;
  const ratio = touchDistance(event) / pinchDistance;
  if (ratio > PINCH_STEP || ratio < 1 / PINCH_STEP) {
    zoomBy(ratio > 1 ? 1 : -1);
    pinchDistance = touchDistance(event);
  }
}

// The mark under the finger or the pointer while the scale is dragged, and under the pointer that rests on it
const dragged = ref<(typeof marks.value)[number] | null>(null);
const hovered = ref<(typeof marks.value)[number] | null>(null);

// The number of loaded photos before each row, which places the rows on the scale like the marks
const photosBefore = computed(() => {
  let count = 0;
  return rows.value.map((row) => {
    const before = count;
    if (row.kind === 'row') count += row.items.length;
    return before;
  });
});

// The position of the view on the scale as the share of the photos above it, so the scale serves as the scrollbar
const viewTop = computed(() => {
  const total = props.periods.reduce((sum, entry) => sum + entry.count, 0);
  if (total === 0) return 0;
  return Math.min(((photosBefore.value[firstVisible.value] ?? 0) / total) * 100, 100);
});

// The short name of the period on top, which the position on the scale carries
const viewLabel = computed(() => {
  const header = currentHeader.value;
  if (!header) return '';
  const level = props.scale ?? 'month';
  return new Intl.DateTimeFormat(locale.value, SCALE_FORMATS[level]).format(header.start);
});
// A drag ends with a click on the mark below, which must not jump a second time
let dragEnded = false;

/** Returns the mark of the scale nearest to the height of the pointer. */
function markAt(event: PointerEvent): (typeof marks.value)[number] | null {
  const scale = event.currentTarget;
  if (!(scale instanceof Element)) return null;
  const box = scale.getBoundingClientRect();
  const share = ((event.clientY - box.top) / box.height) * 100;
  let nearest: (typeof marks.value)[number] | null = null;
  for (const mark of marks.value) {
    if (mark.top <= share) nearest = mark;
  }
  return nearest ?? marks.value[0] ?? null;
}

/** Starts a drag along the scale. */
function onScaleDown(event: PointerEvent): void {
  if (event.currentTarget instanceof Element) {
    event.currentTarget.setPointerCapture(event.pointerId);
  }
  dragged.value = markAt(event);
}

/** Follows a drag along the scale, or names the period under the pointer that rests on it. */
function onScaleMove(event: PointerEvent): void {
  if (dragged.value) {
    dragged.value = markAt(event);
    return;
  }
  if (event.pointerType === 'mouse') hovered.value = markAt(event);
}

/** Jumps to the period the drag ended at. */
function onScaleUp(): void {
  const mark = dragged.value;
  dragged.value = null;
  if (!mark) return;
  dragEnded = true;
  void jumpTo(mark.period);
}

/** Jumps to the period of a mark the keyboard chose, which a drag has done already. */
function onMarkClick(period: string): void {
  if (dragEnded) {
    dragEnded = false;
    return;
  }
  void jumpTo(period);
}

/** The place of a timeline kept in the session storage. */
type RestoredView = {
  count: number;
  top: number;
  zoom: TimelineZoom;
};

/** Returns the key of the place of the timeline in the session storage. */
function storageKey(): string {
  return `gallery.view.${props.restoreKey}`;
}

// The place is kept when the page is left, and taken up again once the first photos are there
onBeforeUnmount(() => {
  if (!props.restoreKey) return;
  const view: RestoredView = {
    count: props.items.length,
    top: scrollArea.value?.$el?.scrollTop ?? 0,
    zoom: zoom.value,
  };
  try {
    sessionStorage.setItem(storageKey(), JSON.stringify(view));
  } catch {
    // Without the storage the timeline starts at the top again
  }
});

// Only a way back through the history returns to the place, a link to the page starts at the top
onMounted(async () => {
  const historyState: unknown = window.history.state;
  if (!props.restoreKey || !isRecord(historyState) || typeof historyState.forward !== 'string') {
    return;
  }
  let view: RestoredView | null = null;
  try {
    view = parseRestoredView(JSON.parse(sessionStorage.getItem(storageKey()) ?? 'null'));
    sessionStorage.removeItem(storageKey());
  } catch {
    return;
  }
  if (!view || !zoomLevels.value.includes(view.zoom)) return;
  zoom.value = view.zoom;
  await props.ensureLoaded(view.count);
  await nextTick();
  if (scrollArea.value?.$el) scrollArea.value.$el.scrollTop = view.top;
});

/** Reads a kept place of a timeline, or returns null for anything else. */
function parseRestoredView(raw: unknown): RestoredView | null {
  if (!isRecord(raw)) return null;
  const { count, top, zoom: level } = raw;
  if (typeof count !== 'number' || typeof top !== 'number' || !isOneOf(TIMELINE_ZOOMS, level)) {
    return null;
  }
  return {
    count,
    top,
    zoom: level,
  };
}
</script>

<template>
  <div class="flex flex-col gap-2" :class="{ 'h-full min-h-0': fill }">
    <div class="flex items-center gap-2">
      <span class="text-highlighted min-w-0 flex-1 truncate text-sm font-semibold">
        {{ pinnedHeader ? headerLabel(pinnedHeader) : '' }}
      </span>
      <UToggleGroup
        type="single"
        :model-value="zoom"
        :items="zoomItems"
        attached
        size="sm"
        color="neutral"
        variant="outline"
        active-color="primary"
        active-variant="soft"
        @update:model-value="onZoomSelect"
      />
    </div>

    <!-- The page itself does not zoom on a pinch, which zooms the timeline instead -->
    <div
      class="relative flex touch-pan-y gap-2"
      :class="{ 'min-h-0 flex-1': fill }"
      :style="pageHeight"
      @wheel="onWheel"
      @keydown="onKeydown"
      @touchstart.passive="onTouchStart"
      @touchmove.passive="onTouchMove"
    >
      <div ref="container" class="min-w-0 flex-1">
        <UScrollArea
          ref="scrollArea"
          :items="rows"
          :virtualize="{
            estimateSize: (index: number) =>
              rows[index]?.kind === 'header' ? HEADER_HEIGHT : tileSize + TIMELINE_GAP,
            overscan: 6,
          }"
          class="h-full scrollbar-none"
        >
          <template #default="{ item: row }">
            <div v-if="row.kind === 'header'" class="group flex items-center gap-2 pt-4 pb-2">
              <h2 class="text-highlighted text-sm font-semibold">{{ headerLabel(row) }}</h2>
              <span class="text-muted text-xs tabular-nums">{{ headerCount(row) }}</span>
              <UCheckbox
                v-if="selectable"
                :model-value="isGroupSelected(row.items)"
                :aria-label="t('gallery.selectPeriod')"
                class="transition-opacity"
                :class="
                  isSelecting ? '' : 'opacity-0 group-hover:opacity-100 focus-within:opacity-100'
                "
                @update:model-value="toggleGroup(row.items)"
              />
            </div>
            <div
              v-else
              class="grid"
              :style="{
                gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
                gap: `${TIMELINE_GAP}px`,
                paddingBottom: `${TIMELINE_GAP}px`,
              }"
            >
              <div
                v-for="item in row.items"
                :key="itemKey(item)"
                :data-tile="positions.get(itemKey(item))"
                class="group relative aspect-square overflow-hidden rounded-sm"
                @click.capture="onTileClick($event, item)"
                @pointerdown="onTilePointerDown($event, item)"
                @pointerup="onTilePointerEnd"
                @pointercancel="onTilePointerEnd"
                @contextmenu="
                  selectable && $event.pointerType === 'touch' && $event.preventDefault()
                "
              >
                <slot name="tile" :item="item" />
                <!-- The frame lies inside the tile, so the edge of the list never cuts it off -->
                <span
                  v-if="selectedKeys.has(itemKey(item))"
                  class="ring-primary bg-primary/10 pointer-events-none absolute inset-0 rounded-sm ring-3 ring-inset"
                ></span>
                <UCheckbox
                  v-if="selectable"
                  :model-value="selectedKeys.has(itemKey(item))"
                  :aria-label="t('gallery.selectPhoto')"
                  class="absolute top-1.5 left-1.5 transition-opacity"
                  :class="
                    isSelecting ? '' : 'opacity-0 group-hover:opacity-100 focus-within:opacity-100'
                  "
                  @click.stop.prevent="choose(item, $event.shiftKey)"
                />
              </div>
            </div>
          </template>
        </UScrollArea>
      </div>

      <!-- The scale of the periods in place of the scrollbar, which shows the position of the view,
           jumps to a period on a click, and follows a drag. Its margin keeps the position at either end
           clear of the zoom above and the edge below. -->
      <nav
        v-if="marks.length > 0"
        class="relative my-3 w-16 shrink-0 cursor-pointer touch-none select-none"
        :aria-label="t('gallery.scale')"
        @pointerdown="onScaleDown"
        @pointermove="onScaleMove"
        @pointerup="onScaleUp"
        @pointerleave="hovered = null"
        @pointercancel="dragged = null"
      >
        <!-- The track of the scale, which the marks sit on -->
        <span
          class="bg-accented pointer-events-none absolute inset-y-0 right-1 w-0.5 rounded-full"
        ></span>
        <button
          v-for="mark in marks"
          :key="mark.period"
          type="button"
          class="group absolute right-0 flex w-full -translate-y-1/2 items-center justify-end gap-1.5"
          :style="{ top: `${mark.top}%` }"
          :aria-label="mark.label"
          @click="onMarkClick(mark.period)"
        >
          <span v-if="mark.parent" class="text-toned truncate text-xs font-medium tabular-nums">{{
            mark.parent
          }}</span>
          <span class="bg-inverted/40 group-hover:bg-primary block h-0.5 w-2.5 rounded-full"></span>
        </button>
        <!-- The position of the view with the period on top, beside the scale so it never covers a mark.
             The period under the pointer takes its place while it shows. -->
        <span
          v-if="!(dragged ?? hovered)"
          class="bg-primary text-inverted pointer-events-none absolute right-full mr-1 -translate-y-1/2 rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap tabular-nums shadow-sm"
          :style="{ top: `${viewTop}%` }"
        >
          {{ viewLabel }}
        </span>
        <!-- The period under the finger or the pointer -->
        <span
          v-if="dragged ?? hovered"
          class="bg-inverted text-inverted pointer-events-none absolute right-full mr-2 -translate-y-1/2 rounded-md px-2 py-1 text-xs whitespace-nowrap"
          :style="{ top: `${(dragged ?? hovered)?.top}%` }"
        >
          {{ (dragged ?? hovered)?.label }}
        </span>
      </nav>
    </div>

    <div v-if="loading" class="flex justify-center py-2">
      <UIcon name="i-lucide-loader-circle" class="text-muted size-5 animate-spin" />
    </div>
  </div>
</template>
