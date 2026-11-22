<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import type { DropdownMenuItem } from '@nuxt/ui';

import { galleryRepository } from '@/api/repositories/gallery';
import type { GalleryRunResponse } from '@/api/types/gallery';
import ImageViewer from '@/components/gallery/ImageViewer.vue';
import { useMutation } from '@/composables/form/useMutation';
import { useDeleteConfirm } from '@/composables/ui/useDeleteConfirm';

// The photos of a run in the image viewer, with or without the overlay for a run that keeps its originals,
// the downloads of the photo and of the run, the deletion of the run, and its details beside the photos.
const props = defineProps<{
  run: GalleryRunResponse | null;
}>();

const open = defineModel<boolean>('open', {
  required: true,
});

const emit = defineEmits<{
  deleted: [run: number];
}>();

const { t, locale } = useI18n();
const { confirmDelete } = useDeleteConfirm();

// The shown photo, counted from 0, -1 while the viewer is closed
const index = ref(-1);

// Shows the captures with the overlay, or the originals without it for a run that has them
const showOverlay = ref(true);

// The positions of the photos of the run, which count from 1 in their addresses
const images = computed(() => Array.from({ length: props.run?.imageCount ?? 0 }, (_, i) => i + 1));
const originalCount = computed(() => props.run?.originalCount ?? 0);

// The time of the first photo in the locale, which the clock of the booth may not know
const takenAt = computed(() =>
  props.run?.takenAt
    ? new Intl.DateTimeFormat(locale.value, {
        dateStyle: 'medium',
        timeStyle: 'short',
      }).format(new Date(props.run.takenAt))
    : t('pages.gallery.takenAtUnknown'),
);

watch(open, (isOpen) => {
  if (!isOpen) {
    index.value = -1;
    return;
  }
  index.value = 0;
  showOverlay.value = true;
});

// The viewer closes itself with -1, which closes the lightbox
watch(index, (value) => {
  if (value < 0) open.value = false;
});

/** Returns the address of the photo shown at a position, the capture or its original. */
function imageUrl(image: number): string {
  if (props.run === null) return '';
  return showOverlay.value
    ? galleryRepository.captureUrl(props.run.run, image)
    : galleryRepository.originalUrl(props.run.run, image);
}

/** Returns the downloads of the shown photo and of the whole run. */
function downloadItems(image: number): DropdownMenuItem[] {
  if (props.run === null) return [];
  const items: DropdownMenuItem[] = [
    {
      label: t('pages.gallery.downloadImage'),
      icon: 'i-lucide-image-down',
      to: imageUrl(image),
      external: true,
      download: true,
    },
    {
      label: t('pages.gallery.downloadZip'),
      icon: 'i-lucide-folder-down',
      to: galleryRepository.zipUrl(props.run.run),
      external: true,
      download: true,
    },
  ];
  if (originalCount.value > 0) {
    items.push({
      label: t('pages.gallery.downloadOriginals'),
      icon: 'i-lucide-folder-down',
      to: galleryRepository.zipUrl(props.run.run, true),
      external: true,
      download: true,
    });
  }
  return items;
}

const { mutate: deleteRun, pending: deleting } = useMutation(
  async () => {
    if (props.run) await galleryRepository.delete(props.run.run);
  },
  {
    confirm: async () =>
      confirmDelete({
        title: t('pages.gallery.deleteConfirmTitle'),
        description: t('pages.gallery.deleteConfirmText'),
      }),
    success: () => t('pages.gallery.deleteSuccess'),
    onSuccess: () => {
      open.value = false;
      if (props.run) emit('deleted', props.run.run);
    },
  },
);
</script>

<template>
  <ImageViewer
    v-if="open && run !== null && index >= 0"
    :key="run.run"
    v-model:index="index"
    :items="images"
    :src="imageUrl"
    :alt="`${t('pages.gallery.run')} ${run.run}`"
  >
    <template #caption>
      <span class="text-on-scrim text-sm font-medium"
        >{{ t('pages.gallery.run') }} #{{ run.run }}</span
      >
      <USwitch
        v-if="originalCount > 0"
        v-model="showOverlay"
        :label="t('pages.gallery.overlay')"
        :ui="{ label: 'text-on-scrim' }"
      />
    </template>
    <template #actions="{ item }">
      <!-- The menu stays inside the viewer, which lies above every portal of the page -->
      <UDropdownMenu :items="downloadItems(item)" :portal="false">
        <UButton
          :label="t('pages.gallery.download')"
          icon="i-lucide-download"
          trailing-icon="i-lucide-chevron-down"
          color="neutral"
          variant="ghost"
          size="sm"
          class="text-on-scrim hover:bg-on-scrim/10"
        />
      </UDropdownMenu>
      <UButton
        :label="t('common.delete')"
        icon="i-lucide-trash-2"
        color="error"
        variant="ghost"
        size="sm"
        :loading="deleting"
        @click="void deleteRun()"
      />
    </template>
    <template #info>
      <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
        <dt class="text-muted">{{ t('pages.gallery.run') }}</dt>
        <dd class="text-highlighted">#{{ run.run }}</dd>
        <dt class="text-muted">{{ t('pages.gallery.event') }}</dt>
        <dd class="text-highlighted">{{ run.event }}</dd>
        <dt class="text-muted">{{ t('pages.gallery.takenAt') }}</dt>
        <dd class="text-highlighted">{{ takenAt }}</dd>
        <dt class="text-muted">{{ t('pages.gallery.photos') }}</dt>
        <dd class="text-highlighted tabular-nums">{{ run.imageCount }}</dd>
        <template v-if="originalCount > 0">
          <dt class="text-muted">{{ t('pages.gallery.originals') }}</dt>
          <dd class="text-highlighted tabular-nums">{{ originalCount }}</dd>
        </template>
      </dl>
    </template>
  </ImageViewer>
</template>
