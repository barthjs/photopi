<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { systemRepository } from '@/api/repositories/system';
import type { SystemInfoResponse } from '@/api/types';
import { useApiError } from '@/composables/useApiError';
import StatCard from '@/components/dashboard/StatCard.vue';
import DocsLink from '@/components/system/DocsLink.vue';

const { t } = useI18n();
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

/** formatBytes shows a size in bytes with a binary unit. */
function formatBytes(bytes: number): string {
  if (bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const value = bytes / 1024 ** exponent;
  return `${value.toFixed(exponent === 0 ? 0 : 1)} ${units[exponent]}`;
}

/** formatUptime shows a duration in seconds as days, hours, and minutes. */
function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);

  const parts: string[] = [];
  if (days) parts.push(`${days}${t('pages.dashboard.daysShort')}`);
  if (days || hours) parts.push(`${hours}${t('pages.dashboard.hoursShort')}`);
  parts.push(`${minutes}${t('pages.dashboard.minutesShort')}`);

  return parts.join(' ');
}

const ipsLabel = computed(() =>
  info.value?.ips.length ? info.value.ips.join(', ') : t('pages.dashboard.noIps'),
);
const cpuLabel = computed(() => (info.value ? `${info.value.cpuPercent.toFixed(1)} %` : ''));
const memLabel = computed(() =>
  info.value ? `${formatBytes(info.value.memUsed)} / ${formatBytes(info.value.memTotal)}` : '',
);
const diskLabel = computed(() =>
  info.value ? `${formatBytes(info.value.diskUsed)} / ${formatBytes(info.value.diskTotal)}` : '',
);
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex items-center gap-1">
      <h1 class="text-highlighted text-xl font-semibold">{{ t('pages.dashboard.title') }}</h1>
      <DocsLink page="dashboard" />
    </div>

    <div v-if="loading" class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <USkeleton v-for="i in 6" :key="i" class="h-20 w-full" />
    </div>

    <UAlert
      v-else-if="loadFailed"
      :title="t('pages.dashboard.loadError')"
      :description="messageFor(loadError)"
      icon="i-lucide-circle-alert"
      color="error"
    />

    <div v-else-if="info" class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <StatCard
        :label="t('pages.dashboard.uptime')"
        :help="t('pages.dashboard.uptimeHint')"
        icon="i-lucide-clock"
        :value="formatUptime(info.uptimeSeconds)"
      />
      <StatCard
        :label="t('pages.dashboard.hostname')"
        icon="i-lucide-server"
        :value="info.hostname"
      />
      <StatCard
        :label="t('pages.dashboard.ipAddresses')"
        :help="t('pages.dashboard.ipAddressesHint')"
        icon="i-lucide-network"
        :value="ipsLabel"
      />
      <StatCard
        :label="t('pages.dashboard.cpu')"
        :help="t('pages.dashboard.cpuHint')"
        icon="i-lucide-cpu"
        :value="cpuLabel"
      />
      <StatCard
        :label="t('pages.dashboard.memory')"
        icon="i-lucide-memory-stick"
        :value="memLabel"
      />
      <StatCard
        :label="t('pages.dashboard.disk')"
        :help="t('pages.dashboard.diskHint')"
        icon="i-lucide-hard-drive"
        :value="diskLabel"
      />
    </div>
  </div>
</template>
