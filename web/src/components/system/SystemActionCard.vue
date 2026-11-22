<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import SystemActionButton from '@/components/system/SystemActionButton.vue';
import { useCopy } from '@/composables/ui/useCopy';

/** An action of the system page, named by the key of its texts. */
export type SystemAction = {
  key: string;
  icon: string;
  color: 'neutral' | 'warning' | 'error';
  // The configured shell command, empty while none is configured, and left out for an action without a command
  command?: string;
  run: () => Promise<void>;
};

// The actions on one target, such as the application or the kiosk, with the commands they run below their buttons
const props = defineProps<{
  title: string;
  actions: SystemAction[];
}>();

const { t } = useI18n();
const { copy } = useCopy();

const hasCommands = computed(() => props.actions.some((action) => action.command !== undefined));
</script>

<template>
  <UCard variant="subtle">
    <template #header>
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h2 class="text-highlighted heading-section">{{ title }}</h2>
        <div class="flex flex-wrap gap-2">
          <SystemActionButton
            v-for="action in actions"
            :key="action.key"
            :label="t(`pages.system.${action.key}.action`)"
            :icon="action.icon"
            :color="action.color"
            :confirm-title="t(`pages.system.${action.key}.confirmTitle`)"
            :confirm-text="t(`pages.system.${action.key}.confirmText`)"
            :success-message="t(`pages.system.${action.key}.success`)"
            :disabled="action.command === ''"
            :run="action.run"
          />
        </div>
      </div>
    </template>

    <!-- Only the actions that run a configured command show it below the header -->
    <template v-if="hasCommands" #default>
      <div class="flex flex-col gap-2">
        <template v-for="action in actions" :key="action.key">
          <div v-if="action.command !== undefined" class="flex min-w-0 items-center gap-2">
            <span class="text-muted w-28 shrink-0 text-sm">
              {{ t(`pages.system.${action.key}.action`) }}
            </span>
            <template v-if="action.command">
              <code
                class="bg-elevated text-default min-w-0 flex-1 truncate rounded-md px-2 py-1 font-mono text-xs"
                :title="action.command"
              >
                {{ action.command }}
              </code>
              <UTooltip :text="t('pages.system.copyCommand')">
                <UButton
                  icon="i-lucide-copy"
                  color="neutral"
                  variant="ghost"
                  size="sm"
                  :aria-label="t('pages.system.copyCommand')"
                  @click="copy(action.command, t('pages.system.commandCopied'))"
                />
              </UTooltip>
            </template>
            <ULink
              v-else
              :to="{
                name: 'config',
                query: {
                  section: 'system',
                },
              }"
              class="text-primary text-sm"
            >
              {{ t('pages.system.notConfigured') }}
            </ULink>
          </div>
        </template>
      </div>
    </template>
  </UCard>
</template>
