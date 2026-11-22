<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import ConfigFieldHint from '@/components/config/ConfigFieldHint.vue';
import { CONFIG_LIMITS, useConfigForm } from '@/composables/config/useConfigForm';
import { countLabel } from '@/lib/form/validation';

// The commands behind the restart and stop actions of the system page
const { t } = useI18n();
const { state, fieldProps } = useConfigForm();
</script>

<template>
  <UCard variant="subtle">
    <div class="flex flex-col gap-6">
      <p class="text-muted text-sm">{{ t('pages.config.system.hint') }}</p>

      <div class="flex flex-col gap-4">
        <h3 class="text-highlighted heading-card">
          {{ t('pages.config.system.sections.server') }}
        </h3>
        <div class="grid grid-cols-1 gap-4">
          <UFormField
            :label="t('fields.system.serverRestartCommand')"
            name="system.serverRestartCommand"
          >
            <template #hint>
              <ConfigFieldHint
                path="system.serverRestartCommand"
                :count="countLabel(state.system.serverRestartCommand, CONFIG_LIMITS.systemCommand)"
              />
            </template>
            <UInput
              v-model="state.system.serverRestartCommand"
              v-bind="fieldProps('system.serverRestartCommand')"
              :placeholder="t('pages.config.system.commandPlaceholder')"
              :maxlength="CONFIG_LIMITS.systemCommand"
              class="w-full font-mono"
            />
          </UFormField>
          <UFormField :label="t('fields.system.serverStopCommand')" name="system.serverStopCommand">
            <template #hint>
              <ConfigFieldHint
                path="system.serverStopCommand"
                :count="countLabel(state.system.serverStopCommand, CONFIG_LIMITS.systemCommand)"
              />
            </template>
            <UInput
              v-model="state.system.serverStopCommand"
              v-bind="fieldProps('system.serverStopCommand')"
              :placeholder="t('pages.config.system.commandPlaceholder')"
              :maxlength="CONFIG_LIMITS.systemCommand"
              class="w-full font-mono"
            />
          </UFormField>
        </div>
      </div>

      <div class="flex flex-col gap-4">
        <h3 class="text-highlighted heading-card">
          {{ t('pages.config.system.sections.kiosk') }}
        </h3>
        <div class="grid grid-cols-1 gap-4">
          <UFormField
            :label="t('fields.system.kioskRestartCommand')"
            name="system.kioskRestartCommand"
          >
            <template #hint>
              <ConfigFieldHint
                path="system.kioskRestartCommand"
                :count="countLabel(state.system.kioskRestartCommand, CONFIG_LIMITS.systemCommand)"
              />
            </template>
            <UInput
              v-model="state.system.kioskRestartCommand"
              v-bind="fieldProps('system.kioskRestartCommand')"
              :placeholder="t('pages.config.system.commandPlaceholder')"
              :maxlength="CONFIG_LIMITS.systemCommand"
              class="w-full font-mono"
            />
          </UFormField>
          <UFormField :label="t('fields.system.kioskStopCommand')" name="system.kioskStopCommand">
            <template #hint>
              <ConfigFieldHint
                path="system.kioskStopCommand"
                :count="countLabel(state.system.kioskStopCommand, CONFIG_LIMITS.systemCommand)"
              />
            </template>
            <UInput
              v-model="state.system.kioskStopCommand"
              v-bind="fieldProps('system.kioskStopCommand')"
              :placeholder="t('pages.config.system.commandPlaceholder')"
              :maxlength="CONFIG_LIMITS.systemCommand"
              class="w-full font-mono"
            />
          </UFormField>
        </div>
      </div>

      <div class="flex flex-col gap-4">
        <h3 class="text-highlighted heading-card">
          {{ t('pages.config.system.sections.both') }}
        </h3>
        <div class="grid grid-cols-1 gap-4">
          <UFormField :label="t('fields.system.allRestartCommand')" name="system.allRestartCommand">
            <template #hint>
              <ConfigFieldHint
                path="system.allRestartCommand"
                :count="countLabel(state.system.allRestartCommand, CONFIG_LIMITS.systemCommand)"
              />
            </template>
            <UInput
              v-model="state.system.allRestartCommand"
              v-bind="fieldProps('system.allRestartCommand')"
              :placeholder="t('pages.config.system.allCommandPlaceholder')"
              :maxlength="CONFIG_LIMITS.systemCommand"
              class="w-full font-mono"
            />
          </UFormField>
          <UFormField :label="t('fields.system.allStopCommand')" name="system.allStopCommand">
            <template #hint>
              <ConfigFieldHint
                path="system.allStopCommand"
                :count="countLabel(state.system.allStopCommand, CONFIG_LIMITS.systemCommand)"
              />
            </template>
            <UInput
              v-model="state.system.allStopCommand"
              v-bind="fieldProps('system.allStopCommand')"
              :placeholder="t('pages.config.system.allCommandPlaceholder')"
              :maxlength="CONFIG_LIMITS.systemCommand"
              class="w-full font-mono"
            />
          </UFormField>
        </div>
      </div>
    </div>
  </UCard>
</template>
