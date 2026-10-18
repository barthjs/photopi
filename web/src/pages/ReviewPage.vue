<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { captureRepository } from '@/api/repositories/capture';
import { useInactivityTimer } from '@/composables/useInactivityTimer';
import { useBoothStore } from '@/stores/booth';

// A guest who walks away leaves the photos to the next guest otherwise. The run stays stored,
// the next guest only gets the welcome screen.
const REVIEW_TIMEOUT_MS = 90_000;

const { t } = useI18n();
const router = useRouter();

const booth = useBoothStore();

const inactivity = useInactivityTimer(() =>
  router.push({
    name: 'welcome',
  }),
);

const showDiscardDialog = ref(false);

const indices = computed(() =>
  Array.from(
    {
      length: booth.imagesPerRun ?? 0,
    },
    (_, i) => i + 1,
  ),
);

onMounted(() => {
  if (booth.run === null || booth.capturedCount !== booth.imagesPerRun) {
    void router.replace({
      name: 'welcome',
    });
    return;
  }

  inactivity.arm(REVIEW_TIMEOUT_MS);
});

/** confirmDiscard discards the run and returns to the welcome screen. */
async function confirmDiscard(): Promise<void> {
  showDiscardDialog.value = false;
  await booth.abort();
  await router.push({
    name: 'welcome',
  });
}

/** onContinue moves on to sharing the run. */
function onContinue(): void {
  void router.push({
    name: 'share',
  });
}

/** openDiscardDialog asks before the run is discarded. */
function openDiscardDialog(): void {
  showDiscardDialog.value = true;
}
</script>

<template>
  <div class="flex flex-1 flex-col gap-6 overflow-hidden p-6">
    <div class="grid flex-1 touch-pan-y snap-y snap-proximity grid-cols-2 gap-4 overflow-y-auto">
      <img
        v-for="index in indices"
        :key="index"
        :src="captureRepository.previewUrl(booth.run as number, index)"
        :alt="`${index}`"
        class="bg-muted h-full w-full snap-start rounded-lg object-cover"
      />
    </div>

    <div class="flex justify-center gap-6">
      <UButton
        :label="t('pages.review.discard')"
        icon="i-lucide-trash-2"
        color="error"
        variant="soft"
        size="xl"
        @click="openDiscardDialog"
        class="h-16 w-56 justify-center text-xl"
      />
      <UButton
        :label="t('pages.review.continueButton')"
        icon="i-lucide-check"
        size="xl"
        @click="onContinue"
        class="h-16 w-56 justify-center text-xl"
      />
    </div>

    <UModal
      v-model:open="showDiscardDialog"
      :title="t('pages.review.discardDialog.title')"
      :description="t('pages.review.discardDialog.text')"
      :ui="{ footer: 'justify-end' }"
    >
      <template #footer="{ close }">
        <UButton :label="t('common.back')" color="neutral" variant="outline" @click="close" />
        <UButton
          :label="t('pages.review.discardDialog.confirm')"
          color="error"
          @click="confirmDiscard"
        />
      </template>
    </UModal>
  </div>
</template>
