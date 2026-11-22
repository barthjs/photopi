<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { useMutation } from '@/composables/form/useMutation';
import { useBoothStore } from '@/stores/booth';
import { useHealthStore } from '@/stores/health';
import { useMetaStore } from '@/stores/meta';

const { t } = useI18n();
const router = useRouter();
const booth = useBoothStore();
const metaStore = useMetaStore();
const health = useHealthStore();

const title = computed(() => metaStore.meta?.capture.welcomeMessage || t('pages.welcome.title'));

// Every run starts without the state of the previous guest
onMounted(() => {
  booth.reset();
});

const { mutate: start, pending } = useMutation(async () => booth.startRun(), {
  onSuccess: () => {
    void router.push({
      name: 'capture',
    });
  },
});
</script>

<template>
  <div class="relative flex flex-1 flex-col items-center justify-center gap-12 px-6 text-center">
    <!-- The welcome message is free text, so the heading scales with the viewport and is cut after
         three lines to fit the screen, which does not scroll. -->
    <h1 class="text-highlighted text-display line-clamp-3 font-semibold">
      {{ title }}
    </h1>

    <UButton
      v-if="health.isReady('camera')"
      size="xl"
      :label="t('pages.welcome.start')"
      :loading="pending"
      class="h-20 w-64 justify-center text-4xl"
      @click="start()"
    />

    <p v-else class="text-error flex items-center gap-2 text-lg font-medium">
      <UIcon name="i-lucide-circle-alert" class="size-5 shrink-0" />
      {{ t('pages.welcome.cameraUnavailable') }}
    </p>

    <UButton
      icon="i-lucide-settings"
      color="neutral"
      variant="link"
      size="xs"
      :aria-label="t('pages.welcome.admin')"
      :to="{ name: 'dashboard' }"
      class="text-muted absolute right-2 bottom-2 opacity-50 hover:opacity-100"
    />

    <UColorModeSwitch class="absolute bottom-2 left-2 opacity-50 hover:opacity-100" />
  </div>
</template>
