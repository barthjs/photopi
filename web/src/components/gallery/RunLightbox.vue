<script setup lang="ts">
import { ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { galleryRepository } from '@/api/repositories/gallery';
import HelpHint from '@/components/form/HelpHint.vue';
import { useMutation } from '@/composables/form/useMutation';

const props = defineProps<{
  run: number | null;
  imageCount: number;
  originalCount: number;
}>();

const emit = defineEmits<{
  deleted: [run: number];
}>();

const open = defineModel<boolean>('open', {
  required: true,
});

const { t } = useI18n();
// Counts from 0 like the select event of the carousel. The shown position adds 1.
const selectedIndex = ref(0);

// Shows the originals without the overlay instead of the captures, for a run that has them
const showOriginals = ref(false);

watch(open, (isOpen) => {
  if (isOpen) {
    selectedIndex.value = 0;
    showOriginals.value = false;
  }
});

/** imageUrl returns the address of the photo shown at a position, the capture or its original. */
function imageUrl(run: number, index: number): string {
  return showOriginals.value
    ? galleryRepository.originalUrl(run, index)
    : galleryRepository.captureUrl(run, index);
}

/** onSelect follows the photo shown in the carousel. */
function onSelect(index: number): void {
  selectedIndex.value = index;
}

const showDeleteConfirm = ref(false);

/** openDeleteConfirm asks before the run is deleted. */
function openDeleteConfirm(): void {
  showDeleteConfirm.value = true;
}

const { mutate: deleteRun, pending: deleting } = useMutation(
  () => galleryRepository.delete(props.run as number),
  {
    success: () => t('pages.gallery.deleteSuccess'),
    onSuccess: () => {
      showDeleteConfirm.value = false;
      open.value = false;
      emit('deleted', props.run as number);
    },
  },
);
</script>

<template>
  <UModal
    v-model:open="open"
    :title="`${t('pages.gallery.run')} #${run}`"
    fullscreen
    :ui="{ body: 'flex-1' }"
  >
    <template #body>
      <div v-if="run !== null" class="flex h-full flex-col items-center justify-center gap-4">
        <!-- Keyed on the run, so the carousel starts at the first photo every time it opens. -->
        <UCarousel
          :key="run"
          :items="Array.from({ length: imageCount }, (_, i) => i + 1)"
          :arrows="imageCount > 1"
          :ui="{
            root: 'h-full w-full',
            viewport: 'h-full',
            container: 'h-full',
            item: 'flex h-full min-w-0 shrink-0 basis-full items-center justify-center',
            // The carousel fills the modal, so the arrows sit inside its edge instead of outside,
            // where they would add a horizontal scrollbar.
            prev: 'start-2 sm:start-2',
            next: 'end-2 sm:end-2',
          }"
          @select="onSelect"
        >
          <template #default="{ item }">
            <img
              :src="imageUrl(run, item)"
              :alt="`${t('pages.gallery.run')} ${run} - ${item}`"
              class="mx-auto max-h-full max-w-full rounded-lg object-contain"
            />
          </template>
        </UCarousel>
        <div class="flex items-center gap-4">
          <p class="text-muted text-sm">{{ selectedIndex + 1 }} / {{ imageCount }}</p>
          <USwitch
            v-if="originalCount > 0"
            v-model="showOriginals"
            :label="t('pages.gallery.showOriginals')"
          />
        </div>
      </div>
    </template>

    <template #footer>
      <div class="flex w-full items-center justify-between gap-2">
        <div class="flex items-center gap-1.5">
          <UButton
            :label="t('common.delete')"
            icon="i-lucide-trash-2"
            color="error"
            variant="outline"
            @click="openDeleteConfirm"
          />
          <HelpHint :text="t('pages.gallery.deleteHint')" />
        </div>
        <div class="flex flex-wrap items-center justify-end gap-1.5">
          <UButton
            v-if="run !== null"
            :label="t('pages.gallery.downloadImage')"
            icon="i-lucide-image-down"
            color="neutral"
            variant="outline"
            :to="imageUrl(run, selectedIndex + 1)"
            download
          />
          <HelpHint :text="t('pages.gallery.downloadZipHint')" />
          <UButton
            :label="t('pages.gallery.downloadZip')"
            icon="i-lucide-download"
            :to="run !== null ? galleryRepository.zipUrl(run) : undefined"
            target="_blank"
          />
          <UButton
            v-if="originalCount > 0"
            :label="t('pages.gallery.downloadOriginals')"
            icon="i-lucide-download"
            color="neutral"
            variant="outline"
            :to="run !== null ? galleryRepository.zipUrl(run, true) : undefined"
            target="_blank"
          />
        </div>
      </div>
    </template>
  </UModal>

  <UModal
    v-model:open="showDeleteConfirm"
    :title="t('pages.gallery.deleteConfirmTitle')"
    :description="t('pages.gallery.deleteConfirmText')"
    :ui="{ footer: 'justify-end' }"
  >
    <template #footer="{ close }">
      <UButton :label="t('common.cancel')" color="neutral" variant="outline" @click="close" />
      <UButton
        :label="t('common.delete')"
        color="error"
        :loading="deleting"
        @click="void deleteRun()"
      />
    </template>
  </UModal>
</template>
