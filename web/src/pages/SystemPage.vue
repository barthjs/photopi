<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useToast } from '@nuxt/ui/composables';

import { systemRepository } from '@/api/repositories/system';
import type { SystemCommandsResponse } from '@/api/types';
import { useApiError } from '@/composables/useApiError';
import HelpHint from '@/components/form/HelpHint.vue';
import SystemActionCard from '@/components/system/SystemActionCard.vue';

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

/** description returns the explanation of a system action, or a note that it is not configured. */
function description(key: keyof SystemCommandsResponse): string {
  return commands.value[key]
    ? t(`pages.system.${key}.description`)
    : t('pages.system.notConfigured');
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <h1 class="text-highlighted text-xl font-semibold">{{ t('pages.system.title') }}</h1>

    <div class="flex flex-col gap-4">
      <div class="flex items-center gap-1.5">
        <h2 class="text-dimmed text-sm font-semibold tracking-wide uppercase">
          {{ t('pages.system.sections.server') }}
        </h2>
        <HelpHint :text="t('pages.system.commandHint')" />
      </div>
      <SystemActionCard
        :title="t('pages.system.serverRestart.title')"
        :description="description('serverRestart')"
        :command="commands.serverRestart"
        :action-label="t('pages.system.serverRestart.action')"
        icon="i-lucide-rotate-cw"
        color="neutral"
        :confirm-title="t('pages.system.serverRestart.confirmTitle')"
        :confirm-text="t('pages.system.serverRestart.confirmText')"
        :success-message="t('pages.system.serverRestart.success')"
        :disabled="!commands.serverRestart"
        :run="() => systemRepository.restartServer()"
      />
      <SystemActionCard
        :title="t('pages.system.serverStop.title')"
        :description="description('serverStop')"
        :command="commands.serverStop"
        :action-label="t('pages.system.serverStop.action')"
        icon="i-lucide-square"
        color="warning"
        :confirm-title="t('pages.system.serverStop.confirmTitle')"
        :confirm-text="t('pages.system.serverStop.confirmText')"
        :success-message="t('pages.system.serverStop.success')"
        :disabled="!commands.serverStop"
        :run="() => systemRepository.stopServer()"
      />
    </div>

    <div class="flex flex-col gap-4">
      <div class="flex items-center gap-1.5">
        <h2 class="text-dimmed text-sm font-semibold tracking-wide uppercase">
          {{ t('pages.system.sections.kiosk') }}
        </h2>
        <HelpHint :text="t('pages.system.commandHint')" />
      </div>
      <SystemActionCard
        :title="t('pages.system.kioskRestart.title')"
        :description="description('kioskRestart')"
        :command="commands.kioskRestart"
        :action-label="t('pages.system.kioskRestart.action')"
        icon="i-lucide-rotate-cw"
        color="neutral"
        :confirm-title="t('pages.system.kioskRestart.confirmTitle')"
        :confirm-text="t('pages.system.kioskRestart.confirmText')"
        :success-message="t('pages.system.kioskRestart.success')"
        :disabled="!commands.kioskRestart"
        :run="() => systemRepository.restartKiosk()"
      />
      <SystemActionCard
        :title="t('pages.system.kioskStop.title')"
        :description="description('kioskStop')"
        :command="commands.kioskStop"
        :action-label="t('pages.system.kioskStop.action')"
        icon="i-lucide-square"
        color="warning"
        :confirm-title="t('pages.system.kioskStop.confirmTitle')"
        :confirm-text="t('pages.system.kioskStop.confirmText')"
        :success-message="t('pages.system.kioskStop.success')"
        :disabled="!commands.kioskStop"
        :run="() => systemRepository.stopKiosk()"
      />
    </div>

    <div class="flex flex-col gap-4">
      <div class="flex items-center gap-1.5">
        <h2 class="text-dimmed text-sm font-semibold tracking-wide uppercase">
          {{ t('pages.system.sections.both') }}
        </h2>
        <HelpHint :text="t('pages.system.commandHint')" />
      </div>
      <SystemActionCard
        :title="t('pages.system.allRestart.title')"
        :description="description('allRestart')"
        :command="commands.allRestart"
        :action-label="t('pages.system.allRestart.action')"
        icon="i-lucide-rotate-cw"
        color="neutral"
        :confirm-title="t('pages.system.allRestart.confirmTitle')"
        :confirm-text="t('pages.system.allRestart.confirmText')"
        :success-message="t('pages.system.allRestart.success')"
        :disabled="!commands.allRestart"
        :run="() => systemRepository.restartAll()"
      />
      <SystemActionCard
        :title="t('pages.system.allStop.title')"
        :description="description('allStop')"
        :command="commands.allStop"
        :action-label="t('pages.system.allStop.action')"
        icon="i-lucide-square"
        color="warning"
        :confirm-title="t('pages.system.allStop.confirmTitle')"
        :confirm-text="t('pages.system.allStop.confirmText')"
        :success-message="t('pages.system.allStop.success')"
        :disabled="!commands.allStop"
        :run="() => systemRepository.stopAll()"
      />
    </div>

    <div class="flex flex-col gap-4">
      <div class="flex items-center gap-1.5">
        <h2 class="text-dimmed text-sm font-semibold tracking-wide uppercase">
          {{ t('pages.system.sections.direct') }}
        </h2>
        <HelpHint :text="t('pages.system.directHint')" />
      </div>
      <SystemActionCard
        :title="t('pages.system.processExit.title')"
        :description="t('pages.system.processExit.description')"
        :action-label="t('pages.system.processExit.action')"
        icon="i-lucide-log-out"
        color="neutral"
        :confirm-title="t('pages.system.processExit.confirmTitle')"
        :confirm-text="t('pages.system.processExit.confirmText')"
        :success-message="t('pages.system.processExit.success')"
        :run="() => systemRepository.exitProcess()"
      />
      <SystemActionCard
        :title="t('pages.system.hostReboot.title')"
        :description="t('pages.system.hostReboot.description')"
        :action-label="t('pages.system.hostReboot.action')"
        icon="i-lucide-refresh-cw"
        color="warning"
        :confirm-title="t('pages.system.hostReboot.confirmTitle')"
        :confirm-text="t('pages.system.hostReboot.confirmText')"
        :success-message="t('pages.system.hostReboot.success')"
        :run="() => systemRepository.rebootHost()"
      />
      <SystemActionCard
        :title="t('pages.system.hostShutdown.title')"
        :description="t('pages.system.hostShutdown.description')"
        :action-label="t('pages.system.hostShutdown.action')"
        icon="i-lucide-power"
        color="error"
        :confirm-title="t('pages.system.hostShutdown.confirmTitle')"
        :confirm-text="t('pages.system.hostShutdown.confirmText')"
        :success-message="t('pages.system.hostShutdown.success')"
        :run="() => systemRepository.shutdownHost()"
      />
    </div>
  </div>
</template>
