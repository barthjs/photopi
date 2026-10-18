<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';

import { publicShareRepository } from '@/api/repositories/publicShare';
import { useApiError } from '@/composables/useApiError';

const { t } = useI18n();
const { messageFor } = useApiError();
const route = useRoute();

const token = computed(() => route.params.token as string);

const loading = ref(true);
const loadFailed = ref(false);
const loadError = ref<unknown>(null);
const imageUrls = ref<string[]>([]);

onMounted(async () => {
  try {
    const info = await publicShareRepository.get(token.value);
    imageUrls.value = info.imageUrls;
  } catch (err) {
    loadFailed.value = true;
    loadError.value = err;
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <div class="mx-auto flex w-full max-w-2xl flex-col gap-4">
    <h1 class="text-highlighted text-xl font-semibold">{{ t('pages.guestShare.title') }}</h1>

    <div v-if="loading" class="grid grid-cols-2 gap-4 sm:grid-cols-3">
      <USkeleton v-for="i in 6" :key="i" class="aspect-square w-full" />
    </div>

    <UAlert
      v-else-if="loadFailed"
      :title="t('pages.guestShare.loadError')"
      :description="messageFor(loadError)"
      icon="i-lucide-circle-alert"
      color="error"
    />

    <template v-else>
      <div class="grid grid-cols-2 gap-4 sm:grid-cols-3">
        <img
          v-for="(url, index) in imageUrls"
          :key="url"
          :src="url"
          :alt="`${index + 1}`"
          loading="lazy"
          decoding="async"
          class="bg-muted aspect-square w-full rounded-lg object-cover"
        />
      </div>

      <UButton
        :label="t('pages.guestShare.downloadZip')"
        icon="i-lucide-download"
        :to="publicShareRepository.zipUrl(token)"
        target="_blank"
        block
      />
    </template>
  </div>
</template>
