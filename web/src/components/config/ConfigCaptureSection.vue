<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import ConfigFieldHint from '@/components/config/ConfigFieldHint.vue';
import { CONFIG_LIMITS, useConfigForm } from '@/composables/config/useConfigForm';
import { countLabel } from '@/lib/form/validation';

// The course of a run and the texts guests see
const { t } = useI18n();
const { state, fieldProps } = useConfigForm();
</script>

<template>
  <UCard variant="subtle">
    <div class="flex flex-col gap-4">
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <UFormField :label="t('fields.capture.eventName')" name="capture.eventName" required>
          <template #hint>
            <ConfigFieldHint
              path="capture.eventName"
              :help="t('pages.config.capture.eventNameHint')"
              :count="countLabel(state.capture.eventName, CONFIG_LIMITS.captureEventName)"
            />
          </template>
          <UInput
            v-model="state.capture.eventName"
            v-bind="fieldProps('capture.eventName')"
            :maxlength="CONFIG_LIMITS.captureEventName"
            class="w-full"
          />
        </UFormField>
        <UFormField :label="t('fields.capture.welcomeMessage')" name="capture.welcomeMessage">
          <template #hint>
            <ConfigFieldHint
              path="capture.welcomeMessage"
              :count="countLabel(state.capture.welcomeMessage, CONFIG_LIMITS.captureWelcomeMessage)"
            />
          </template>
          <UInput
            v-model="state.capture.welcomeMessage"
            v-bind="fieldProps('capture.welcomeMessage')"
            :placeholder="t('pages.welcome.title')"
            :maxlength="CONFIG_LIMITS.captureWelcomeMessage"
            class="w-full"
          />
        </UFormField>
        <UFormField
          :label="t('fields.capture.adminEmail')"
          name="capture.adminEmail"
          class="sm:col-span-2"
        >
          <template #hint>
            <ConfigFieldHint
              path="capture.adminEmail"
              :help="t('pages.config.capture.adminEmailHint')"
            />
          </template>
          <UInput
            v-model="state.capture.adminEmail"
            v-bind="fieldProps('capture.adminEmail')"
            type="email"
            class="w-full"
          />
        </UFormField>
      </div>
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <UFormField :label="t('fields.capture.imagesPerRun')" name="capture.imagesPerRun">
          <template #hint>
            <ConfigFieldHint path="capture.imagesPerRun" />
          </template>
          <UInputNumber
            v-model="state.capture.imagesPerRun"
            v-bind="fieldProps('capture.imagesPerRun')"
            :min="1"
            :max="20"
            class="w-full"
          />
        </UFormField>
        <UFormField :label="t('fields.capture.filePrefix')" name="capture.filePrefix" required>
          <template #hint>
            <ConfigFieldHint
              path="capture.filePrefix"
              :count="countLabel(state.capture.filePrefix, CONFIG_LIMITS.captureFilePrefix)"
            />
          </template>
          <UInput
            v-model="state.capture.filePrefix"
            v-bind="fieldProps('capture.filePrefix')"
            :maxlength="CONFIG_LIMITS.captureFilePrefix"
            class="w-full"
          />
        </UFormField>
        <UFormField :label="t('fields.capture.saveOriginals')" name="capture.saveOriginals">
          <template #label="{ label }">
            <span class="inline-flex items-center gap-1.5">
              {{ label }}
              <ConfigFieldHint
                path="capture.saveOriginals"
                :help="t('pages.config.capture.saveOriginalsHint')"
              />
            </span>
          </template>
          <USwitch
            v-model="state.capture.saveOriginals"
            v-bind="fieldProps('capture.saveOriginals')"
          />
        </UFormField>
      </div>
    </div>
  </UCard>
</template>
