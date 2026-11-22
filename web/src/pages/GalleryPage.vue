<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import type { DropdownMenuItem } from '@nuxt/ui';

import { galleryRepository } from '@/api/repositories/gallery';
import type { GalleryEventResponse, GalleryRunResponse } from '@/api/types/gallery';
import PhotoTile from '@/components/gallery/PhotoTile.vue';
import PhotoTimeline from '@/components/gallery/PhotoTimeline.vue';
import RunLightbox from '@/components/gallery/RunLightbox.vue';
import SelectionBar from '@/components/gallery/SelectionBar.vue';
import DocsLink from '@/components/system/DocsLink.vue';
import { useApiError } from '@/composables/form/useApiError';
import { useMutation } from '@/composables/form/useMutation';
import { useTimelineList } from '@/composables/gallery/useTimelineList';
import { useListQuery } from '@/composables/list/useListQuery';
import { useDeleteConfirm } from '@/composables/ui/useDeleteConfirm';
import { useUrlSection } from '@/composables/ui/useUrlSection';
import { toggleKey } from '@/lib/gallery/selection';
import type { TimelineZoom } from '@/lib/gallery/timeline';

const { t } = useI18n();
const { messageFor } = useApiError();
const { confirmDelete } = useDeleteConfirm();

/** The number of runs one page of the gallery loads. */
const PAGE_SIZE = 60;

/** The view of the runs, a timeline of tiles or a list. */
type ViewMode = 'tiles' | 'list';

const events = ref<GalleryEventResponse[]>([]);
const eventsError = ref<unknown>(null);

// The event of the booth, or the most recent one without runs under it, which the gallery opens with
const defaultEvent = computed(
  () => events.value.find((event) => event.current)?.event ?? events.value[0]?.event,
);

const { filters, setFilter, applyDefaultFilters } = useListQuery<{ event?: string }>({
  filterTypes: {
    event: 'string',
  },
  defaultSort: {
    column: 'run',
    direction: 'desc',
  },
  defaultFilters: () => ({
    event: defaultEvent.value,
  }),
});

// The view lives in the URL next to the event, which both views share
const view = useUrlSection<ViewMode>(['tiles', 'list'], 'tiles', ['event']);

const selectedEvent = computed<string | undefined>({
  get: () => filters.value.event,
  set: (value) => setFilter('event', value),
});
const currentEvent = computed(() =>
  events.value.find((event) => event.event === selectedEvent.value),
);
const eventItems = computed(() =>
  events.value.map((event) => ({
    label: event.current ? `${event.event} (${t('pages.gallery.currentEvent')})` : event.event,
    value: event.event,
  })),
);

// The downloads of every photo of the event
const eventDownloadItems = computed<DropdownMenuItem[]>(() => {
  const event = currentEvent.value;
  if (!event) return [];
  return downloadItems(
    (originals) => galleryRepository.eventZipUrl(event.event, originals),
    event.originalCount > 0,
  );
});

/** Returns the download of the photos and, if there are any, of their originals. */
function downloadItems(
  url: (originals: boolean) => string,
  hasOriginals: boolean,
): DropdownMenuItem[] {
  const items: DropdownMenuItem[] = [
    {
      label: t('pages.gallery.downloadZip'),
      icon: 'i-lucide-folder-down',
      to: url(false),
      external: true,
      download: true,
    },
  ];
  if (hasOriginals) {
    items.push({
      label: t('pages.gallery.downloadOriginals'),
      icon: 'i-lucide-folder-down',
      to: url(true),
      external: true,
      download: true,
    });
  }
  return items;
}

const timeline = useTimelineList<GalleryRunResponse>({
  pageSize: PAGE_SIZE,
  enabled: () => selectedEvent.value !== undefined,
  fetch: async (page) =>
    galleryRepository.list({
      filters: {
        event: selectedEvent.value,
      },
      ...page,
    }),
  fetchPeriods: async () =>
    (
      await galleryRepository.listDays({
        event: selectedEvent.value,
      })
    ).days.map((entry) => ({
      period: entry.day,
      count: entry.count,
    })),
});

/** Reads the events and opens the event of the booth while the URL names none. */
async function loadEvents(): Promise<void> {
  eventsError.value = null;
  try {
    events.value = await galleryRepository.listEvents();
    applyDefaultFilters();
  } catch (err) {
    eventsError.value = err;
  }
}

onMounted(async () => {
  await loadEvents();
  await timeline.reload();
});
watch(selectedEvent, () => {
  selected.value = [];
  void timeline.reload();
});

const zoom = ref<TimelineZoom>('day');

/** Returns the time of the first photo of a run, the start of the epoch while it is unknown. */
function takenAt(run: GalleryRunResponse): Date {
  return new Date(run.takenAt ?? 0);
}

const activeRun = ref<GalleryRunResponse | null>(null);
const lightboxOpen = ref(false);

/** Shows a run in the lightbox. */
function openRun(run: GalleryRunResponse): void {
  activeRun.value = run;
  lightboxOpen.value = true;
}

/** Takes a deleted run out of the gallery and reloads the counts of the events. */
function onDeleted(run: number): void {
  timeline.remove((entry) => entry.run === run);
  void loadEvents();
}

// The runs chosen for an action on several of them
const selected = ref<string[]>([]);
const selectedRuns = computed(() => selected.value.map(Number));

// The downloads of the chosen runs, with their originals if the event keeps any
const selectionDownloadItems = computed(() =>
  downloadItems(
    (originals) => galleryRepository.selectionZipUrl(selectedRuns.value, originals),
    (currentEvent.value?.originalCount ?? 0) > 0,
  ),
);

/** Chooses a run in the list view, or leaves it out again. */
function toggleRun(run: GalleryRunResponse): void {
  selected.value = toggleKey(selected.value, String(run.run));
}

const { mutate: deleteSelected, pending: deleting } = useMutation(
  async (ids: string[]) =>
    galleryRepository.bulk({
      action: 'delete',
      ids,
    }),
  {
    confirm: async (ids) =>
      confirmDelete({
        title: t(
          'pages.gallery.deleteSelectedConfirmTitle',
          {
            count: ids.length,
          },
          ids.length,
        ),
        description: t('pages.gallery.deleteConfirmText'),
      }),
    success: (result) =>
      t(
        'pages.gallery.deleteSelectedSuccess',
        {
          count: result.succeeded,
        },
        result.succeeded,
      ),
    onSuccess: async () => {
      selected.value = [];
      await Promise.all([loadEvents(), timeline.reload()]);
    },
  },
);
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-1.5">
        <h1 class="text-highlighted heading-page">{{ t('pages.gallery.title') }}</h1>
        <DocsLink page="gallery" />
      </div>
      <div class="flex w-full flex-wrap items-center gap-2 sm:w-auto">
        <USelect
          v-if="events.length"
          v-model="selectedEvent"
          :items="eventItems"
          icon="i-lucide-calendar"
          class="min-w-0 flex-1 sm:w-64 sm:flex-none"
        />
        <UDropdownMenu v-if="currentEvent" :items="eventDownloadItems">
          <UButton
            :label="t('pages.gallery.download')"
            icon="i-lucide-download"
            trailing-icon="i-lucide-chevron-down"
            color="neutral"
            variant="outline"
          />
        </UDropdownMenu>
        <UFieldGroup>
          <UButton
            icon="i-lucide-layout-grid"
            :color="view === 'tiles' ? 'primary' : 'neutral'"
            :variant="view === 'tiles' ? 'solid' : 'outline'"
            :aria-label="t('pages.gallery.viewTiles')"
            @click="view = 'tiles'"
          />
          <UButton
            icon="i-lucide-list"
            :color="view === 'list' ? 'primary' : 'neutral'"
            :variant="view === 'list' ? 'solid' : 'outline'"
            :aria-label="t('pages.gallery.viewList')"
            @click="view = 'list'"
          />
        </UFieldGroup>
      </div>
    </div>

    <UAlert
      v-if="eventsError || timeline.errorMessage.value"
      :title="t('pages.gallery.loadError')"
      :description="timeline.errorMessage.value ?? messageFor(eventsError)"
      icon="i-lucide-circle-alert"
      color="error"
    />

    <div
      v-else-if="timeline.isLoading.value && timeline.items.value.length === 0"
      class="grid grid-cols-2 gap-1 sm:grid-cols-4 lg:grid-cols-6"
    >
      <USkeleton v-for="i in 12" :key="i" class="aspect-square w-full rounded-sm" />
    </div>

    <UEmpty
      v-else-if="timeline.items.value.length === 0"
      icon="i-lucide-images"
      :description="t('pages.gallery.empty')"
    />

    <PhotoTimeline
      v-else-if="view === 'tiles'"
      v-model:zoom="zoom"
      v-model:selected="selected"
      :items="timeline.items.value"
      :item-key="(run: GalleryRunResponse) => String(run.run)"
      :item-date="takenAt"
      :periods="timeline.periods.value"
      scale="day"
      :zooms="['year', 'month', 'week', 'day']"
      :has-more="timeline.hasMore.value"
      :loading="timeline.isLoading.value"
      :ensure-loaded="timeline.ensureLoaded"
      selectable
      restore-key="gallery"
      @load-more="timeline.loadMore()"
    >
      <template #tile="{ item: run }">
        <PhotoTile
          :src="galleryRepository.thumbnailUrl(run.run)"
          :alt="`${t('pages.gallery.run')} ${run.run}`"
          :caption="`#${run.run}`"
          @open="openRun(run)"
        >
          <template #badges>
            <UBadge
              v-if="run.imageCount > 1"
              icon="i-lucide-layers"
              :label="String(run.imageCount)"
              :aria-label="t('pages.gallery.imageCount', { count: run.imageCount }, run.imageCount)"
              color="neutral"
              variant="solid"
              size="sm"
            />
          </template>
        </PhotoTile>
      </template>
    </PhotoTimeline>

    <template v-else>
      <div class="divide-default border-default divide-y rounded-lg border">
        <div
          v-for="run in timeline.items.value"
          :key="run.run"
          class="hover:bg-elevated flex items-center gap-4 px-4 py-2 transition"
        >
          <UCheckbox
            :model-value="selected.includes(String(run.run))"
            :aria-label="t('gallery.selectPhoto')"
            @update:model-value="toggleRun(run)"
          />
          <button
            type="button"
            class="flex min-w-0 flex-1 items-center gap-4 text-left"
            @click="selected.length > 0 ? toggleRun(run) : openRun(run)"
          >
            <img
              :src="galleryRepository.thumbnailUrl(run.run)"
              :alt="`${t('pages.gallery.run')} ${run.run}`"
              loading="lazy"
              decoding="async"
              class="size-12 shrink-0 rounded-md object-cover"
            />
            <span class="text-highlighted font-medium">#{{ run.run }}</span>
            <span class="text-muted text-sm">
              {{ t('pages.gallery.imageCount', { count: run.imageCount }, run.imageCount) }}
            </span>
          </button>
        </div>
      </div>
      <div v-if="timeline.hasMore.value" class="flex justify-center">
        <UButton
          :label="t('pages.gallery.loadMore')"
          color="neutral"
          variant="soft"
          :loading="timeline.isLoading.value"
          @click="timeline.loadMore()"
        />
      </div>
    </template>

    <SelectionBar :count="selected.length" @clear="selected = []">
      <UDropdownMenu :items="selectionDownloadItems">
        <UButton
          :label="t('pages.gallery.download')"
          icon="i-lucide-download"
          trailing-icon="i-lucide-chevron-down"
          color="neutral"
          variant="ghost"
        />
      </UDropdownMenu>
      <UButton
        :label="t('common.delete')"
        icon="i-lucide-trash-2"
        color="error"
        variant="soft"
        :loading="deleting"
        @click="deleteSelected(selected)"
      />
    </SelectionBar>

    <RunLightbox v-model:open="lightboxOpen" :run="activeRun" @deleted="onDeleted" />
  </div>
</template>
