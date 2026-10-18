<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { galleryRepository } from '@/api/repositories/gallery';
import type { GalleryEventResponse, GalleryRunResponse } from '@/api/types';
import { useApiError } from '@/composables/useApiError';
import HelpHint from '@/components/form/HelpHint.vue';
import RunLightbox from '@/components/gallery/RunLightbox.vue';

const { t } = useI18n();
const { messageFor } = useApiError();

const PAGE_SIZE = 24;
const VIEW_STORAGE_KEY = 'photopi-admin-gallery-view';

type ViewMode = 'tiles' | 'list';

/** initialView returns the view chosen last time, or the tiles. */
function initialView(): ViewMode {
  return localStorage.getItem(VIEW_STORAGE_KEY) === 'list' ? 'list' : 'tiles';
}

const view = ref<ViewMode>(initialView());
watch(view, (value) => localStorage.setItem(VIEW_STORAGE_KEY, value));

/** setView switches between tiles and list. */
function setView(mode: ViewMode): void {
  view.value = mode;
}

const events = ref<GalleryEventResponse[]>([]);
const selectedEvent = ref<string | undefined>(undefined);
const currentEvent = computed(() => events.value.find((e) => e.event === selectedEvent.value));
const eventItems = computed(() =>
  events.value.map((e) => ({
    label: e.current ? `${e.event} (${t('pages.gallery.currentEvent')})` : e.event,
    value: e.event,
  })),
);

const page = ref(1);
const runs = ref<GalleryRunResponse[]>([]);
const totalRuns = ref(0);
const loading = ref(true);
const loadFailed = ref(false);
const loadError = ref<unknown>(null);

/**
 * loadEvents reads the events and selects the event of the booth,
 * or the most recent one if the booth has no runs under its event yet.
 */
async function loadEvents(): Promise<void> {
  loading.value = true;
  loadFailed.value = false;
  loadError.value = null;
  try {
    events.value = await galleryRepository.listEvents();
    selectedEvent.value = events.value.find((e) => e.current)?.event ?? events.value[0]?.event;
    if (selectedEvent.value === undefined) {
      runs.value = [];
      totalRuns.value = 0;
      return;
    }
    await load();
  } catch (err) {
    loadFailed.value = true;
    loadError.value = err;
  } finally {
    loading.value = false;
  }
}

/** load reads the current page of runs of the selected event. */
async function load(): Promise<void> {
  if (selectedEvent.value === undefined) return;

  loading.value = true;
  loadFailed.value = false;
  loadError.value = null;
  try {
    const result = await galleryRepository.list({
      filters: {
        event: selectedEvent.value,
      },
      limit: PAGE_SIZE,
      offset: (page.value - 1) * PAGE_SIZE,
    });
    runs.value = result.data;
    totalRuns.value = result.meta.totalCount;
  } catch (err) {
    loadFailed.value = true;
    loadError.value = err;
  } finally {
    loading.value = false;
  }
}

onMounted(loadEvents);
watch(page, load);
watch(selectedEvent, (event, previous) => {
  // The first selection comes from loadEvents, which loads the runs itself
  if (previous === undefined || event === undefined) return;
  if (page.value !== 1) {
    page.value = 1;
    return;
  }
  void load();
});

const activeRun = ref<GalleryRunResponse | null>(null);
const lightboxOpen = ref(false);

/** openRun shows a run in the lightbox. */
function openRun(run: GalleryRunResponse): void {
  activeRun.value = run;
  lightboxOpen.value = true;
}

/** onDeleted removes a deleted run from the shown page and from the counts of its event. */
function onDeleted(run: number): void {
  const deleted = runs.value.find((r) => r.run === run);
  runs.value = runs.value.filter((r) => r.run !== run);
  totalRuns.value = Math.max(0, totalRuns.value - 1);

  const event = currentEvent.value;
  if (deleted && event) {
    event.runCount -= 1;
    event.imageCount -= deleted.imageCount;
    event.originalCount -= deleted.originalCount;
  }
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-1.5">
        <h1 class="text-highlighted text-xl font-semibold">{{ t('pages.gallery.title') }}</h1>
        <HelpHint :text="t('pages.gallery.hint')" />
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <USelect
          v-if="events.length"
          v-model="selectedEvent"
          :items="eventItems"
          icon="i-lucide-calendar"
          class="w-64"
        />
        <UButton
          v-if="currentEvent"
          :label="t('pages.gallery.downloadEvent')"
          icon="i-lucide-download"
          color="neutral"
          variant="outline"
          :to="galleryRepository.eventZipUrl(currentEvent.event)"
          target="_blank"
        />
        <UButton
          v-if="currentEvent && currentEvent.originalCount > 0"
          :label="t('pages.gallery.downloadEventOriginals')"
          icon="i-lucide-download"
          color="neutral"
          variant="outline"
          :to="galleryRepository.eventZipUrl(currentEvent.event, true)"
          target="_blank"
        />
        <UFieldGroup>
          <UButton
            icon="i-lucide-layout-grid"
            :color="view === 'tiles' ? 'primary' : 'neutral'"
            :variant="view === 'tiles' ? 'solid' : 'outline'"
            :aria-label="t('pages.gallery.viewTiles')"
            @click="setView('tiles')"
          />
          <UButton
            icon="i-lucide-list"
            :color="view === 'list' ? 'primary' : 'neutral'"
            :variant="view === 'list' ? 'solid' : 'outline'"
            :aria-label="t('pages.gallery.viewList')"
            @click="setView('list')"
          />
        </UFieldGroup>
      </div>
    </div>

    <div v-if="loading" class="grid grid-cols-2 gap-4 sm:grid-cols-4 lg:grid-cols-6">
      <USkeleton v-for="i in 12" :key="i" class="aspect-square w-full" />
    </div>

    <UAlert
      v-else-if="loadFailed"
      :title="t('pages.gallery.loadError')"
      :description="messageFor(loadError)"
      icon="i-lucide-circle-alert"
      color="error"
    />

    <p v-else-if="!runs.length" class="text-muted py-12 text-center text-sm">
      {{ t('pages.gallery.empty') }}
    </p>

    <template v-else>
      <div v-if="view === 'tiles'" class="grid grid-cols-2 gap-4 sm:grid-cols-4 lg:grid-cols-6">
        <button
          v-for="run in runs"
          :key="run.run"
          type="button"
          class="group border-default hover:ring-primary relative aspect-square overflow-hidden rounded-lg border ring-2 ring-transparent transition"
          @click="openRun(run)"
        >
          <img
            :src="galleryRepository.captureUrl(run.run, 1)"
            :alt="`${t('pages.gallery.run')} ${run.run}`"
            loading="lazy"
            decoding="async"
            class="size-full object-cover"
          />
          <div
            class="bg-scrim/60 text-on-scrim absolute inset-x-0 bottom-0 flex items-center justify-between px-2 py-1 text-xs"
          >
            <span>#{{ run.run }}</span>
            <span>{{
              t('pages.gallery.imageCount', { count: run.imageCount }, run.imageCount)
            }}</span>
          </div>
        </button>
      </div>

      <div v-else class="divide-default border-default divide-y rounded-lg border">
        <button
          v-for="run in runs"
          :key="run.run"
          type="button"
          class="hover:bg-elevated flex w-full items-center gap-4 px-4 py-2 text-left transition"
          @click="openRun(run)"
        >
          <img
            :src="galleryRepository.captureUrl(run.run, 1)"
            :alt="`${t('pages.gallery.run')} ${run.run}`"
            loading="lazy"
            decoding="async"
            class="size-12 shrink-0 rounded object-cover"
          />
          <span class="text-highlighted font-medium">#{{ run.run }}</span>
          <span class="text-muted text-sm">
            {{ t('pages.gallery.imageCount', { count: run.imageCount }, run.imageCount) }}
          </span>
        </button>
      </div>

      <div v-if="totalRuns > PAGE_SIZE" class="flex justify-center">
        <UPagination v-model:page="page" :total="totalRuns" :items-per-page="PAGE_SIZE" />
      </div>
    </template>

    <RunLightbox
      v-model:open="lightboxOpen"
      :run="activeRun?.run ?? null"
      :image-count="activeRun?.imageCount ?? 0"
      :original-count="activeRun?.originalCount ?? 0"
      @deleted="onDeleted"
    />
  </div>
</template>
