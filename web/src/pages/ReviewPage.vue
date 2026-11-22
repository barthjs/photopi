<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { captureRepository } from '@/api/repositories/capture';
import { useInactivityTimer } from '@/composables/timer/useInactivityTimer';
import { useConfirm } from '@/composables/ui/useConfirm';
import { useDeleteConfirm } from '@/composables/ui/useDeleteConfirm';
import { useBoothStore } from '@/stores/booth';

// A guest who walks away leaves the photos to the next guest otherwise. The run stays stored,
// the next guest only gets the welcome screen.
const REVIEW_TIMEOUT_MS = 90_000;

const { t } = useI18n();
const router = useRouter();

const booth = useBoothStore();
const { settle } = useConfirm();
const { confirmDelete } = useDeleteConfirm();

const inactivity = useInactivityTimer(
  () =>
    void router.push({
      name: 'welcome',
    }),
);

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

// A timeout leaves the page while the dialog may still be open, which only belongs to this page
onBeforeUnmount(() => settle(false));

/** Discards the run after a confirmation and returns to the welcome screen. */
async function discard(): Promise<void> {
  const confirmed = await confirmDelete({
    title: t('pages.review.discardDialog.title'),
    description: t('pages.review.discardDialog.text'),
  });
  if (!confirmed) return;

  await booth.abort();
  await router.push({
    name: 'welcome',
  });
}

/** Moves on to sharing the run. */
function onContinue(): void {
  void router.push({
    name: 'share',
  });
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
        class="h-16 w-56 justify-center text-xl"
        @click="discard"
      />
      <UButton
        :label="t('pages.review.continueButton')"
        icon="i-lucide-check"
        size="xl"
        class="h-16 w-56 justify-center text-xl"
        @click="onContinue"
      />
    </div>
  </div>
</template>
