<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { systemRepository } from '@/api/repositories/system';
import type { SystemInfoResponse } from '@/api/types/system';
import { useApiError } from '@/composables/form/useApiError';
import StatCard from '@/components/dashboard/StatCard.vue';
import DocsLink from '@/components/system/DocsLink.vue';
import { formatBytes } from '@/lib/ui/bytes';

const { t, locale } = useI18n();
const { messageFor } = useApiError();

const info = ref<SystemInfoResponse | null>(null);
const loading = ref(true);
const loadFailed = ref(false);
const loadError = ref<unknown>(null);

onMounted(async () => {
  try {
    info.value = await systemRepository.getInfo();
  } catch (err) {
    loadFailed.value = true;
    loadError.value = err;
  } finally {
    loading.value = false;
  }
});

/** Shows a duration in seconds as days, hours, and minutes. */
function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);

  const parts: string[] = [];
  if (days) parts.push(`${days} ${t('pages.dashboard.daysShort')}`);
  if (days || hours) parts.push(`${hours} ${t('pages.dashboard.hoursShort')}`);
  parts.push(`${minutes} ${t('pages.dashboard.minutesShort')}`);

  return parts.join(' ');
}

const ipsLabel = computed(() =>
  info.value && info.value.ips.length > 0 ? info.value.ips.join(', ') : t('pages.dashboard.noIps'),
);
const cpuLabel = computed(() => (info.value ? `${info.value.cpuPercent.toFixed(1)} %` : ''));
const memLabel = computed(() =>
  info.value
    ? `${formatBytes(info.value.memUsed, locale.value)} / ${formatBytes(info.value.memTotal, locale.value)}`
    : '',
);
const diskLabel = computed(() =>
  info.value
    ? `${formatBytes(info.value.diskUsed, locale.value)} / ${formatBytes(info.value.diskTotal, locale.value)}`
    : '',
);
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex items-center gap-1">
      <h1 class="text-highlighted heading-page">{{ t('pages.dashboard.title') }}</h1>
      <DocsLink page="dashboard" />
    </div>

    <div v-if="loading" class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <USkeleton v-for="i in 5" :key="i" class="h-20 w-full first:col-span-full" />
    </div>

    <UAlert
      v-else-if="loadFailed"
      :title="t('pages.dashboard.loadError')"
      :description="messageFor(loadError)"
      icon="i-lucide-circle-alert"
      color="error"
    />

    <div v-else-if="info" class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <!-- The host with its addresses takes the full width above the measures -->
      <StatCard
        :label="t('pages.dashboard.hostname')"
        text-value
        icon="i-lucide-network"
        :value="info.hostname"
        :detail="ipsLabel"
        class="col-span-full"
      />
      <StatCard
        :label="t('pages.dashboard.uptime')"
        icon="i-lucide-clock"
        :value="formatUptime(info.uptimeSeconds)"
      />
      <StatCard :label="t('pages.dashboard.cpu')" icon="i-lucide-cpu" :value="cpuLabel" />
      <StatCard
        :label="t('pages.dashboard.memory')"
        text-value
        icon="i-lucide-memory-stick"
        :value="memLabel"
      />
      <StatCard
        :label="t('pages.dashboard.disk')"
        text-value
        icon="i-lucide-hard-drive"
        :value="diskLabel"
      />
    </div>
  </div>
</template>
