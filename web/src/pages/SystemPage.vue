<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useToast } from '@nuxt/ui/composables';

import { systemRepository } from '@/api/repositories/system';
import type { SystemCommandsResponse } from '@/api/types/system';
import { useApiError } from '@/composables/form/useApiError';
import DocsLink from '@/components/system/DocsLink.vue';
import SystemActionCard, { type SystemAction } from '@/components/system/SystemActionCard.vue';

const { t } = useI18n();
const { messageFor } = useApiError();
const toast = useToast();

// The configured commands of the actions, an empty one is not configured and disables its action
const commands = ref<SystemCommandsResponse>({
  serverRestart: '',
  serverStop: '',
  kioskRestart: '',
  kioskStop: '',
  allRestart: '',
  allStop: '',
});

onMounted(async () => {
  try {
    const info = await systemRepository.getInfo();
    commands.value = info.commands;
  } catch (err) {
    toast.add({
      title: messageFor(err),
      icon: 'i-lucide-circle-alert',
      color: 'error',
    });
  }
});

/** Returns the restart and the stop of a target, which run the configured commands. */
function commandActions(
  restart: keyof SystemCommandsResponse,
  stop: keyof SystemCommandsResponse,
  runRestart: () => Promise<void>,
  runStop: () => Promise<void>,
): SystemAction[] {
  return [
    {
      key: restart,
      icon: 'i-lucide-rotate-cw',
      color: 'neutral',
      command: commands.value[restart],
      run: runRestart,
    },
    {
      key: stop,
      icon: 'i-lucide-square',
      color: 'warning',
      command: commands.value[stop],
      run: runStop,
    },
  ];
}

// The actions that need no command first, then the targets of the configured commands
const groups = computed(() => [
  {
    title: t('pages.system.sections.direct'),
    actions: [
      {
        key: 'processExit',
        icon: 'i-lucide-log-out',
        color: 'neutral',
        run: async () => systemRepository.exitProcess(),
      },
      {
        key: 'hostReboot',
        icon: 'i-lucide-refresh-cw',
        color: 'warning',
        run: async () => systemRepository.rebootHost(),
      },
      {
        key: 'hostShutdown',
        icon: 'i-lucide-power',
        color: 'error',
        run: async () => systemRepository.shutdownHost(),
      },
    ] satisfies SystemAction[],
  },
  {
    title: t('pages.system.sections.server'),
    actions: commandActions(
      'serverRestart',
      'serverStop',
      async () => systemRepository.restartServer(),
      async () => systemRepository.stopServer(),
    ),
  },
  {
    title: t('pages.system.sections.kiosk'),
    actions: commandActions(
      'kioskRestart',
      'kioskStop',
      async () => systemRepository.restartKiosk(),
      async () => systemRepository.stopKiosk(),
    ),
  },
  {
    title: t('pages.system.sections.both'),
    actions: commandActions(
      'allRestart',
      'allStop',
      async () => systemRepository.restartAll(),
      async () => systemRepository.stopAll(),
    ),
  },
]);
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex items-center gap-1">
      <h1 class="text-highlighted heading-page">{{ t('pages.system.title') }}</h1>
      <DocsLink page="system" />
    </div>

    <SystemActionCard
      v-for="group in groups"
      :key="group.title"
      :title="group.title"
      :actions="group.actions"
    />
  </div>
</template>
