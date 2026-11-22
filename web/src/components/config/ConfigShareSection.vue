<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { configRepository } from '@/api/repositories/config';
import type { CloudShareDriver } from '@/api/types/config';
import ConfigFieldHint from '@/components/config/ConfigFieldHint.vue';
import SecretField from '@/components/form/SecretField.vue';
import { CONFIG_LIMITS, useConfigForm } from '@/composables/config/useConfigForm';
import { useConnectionTestReport } from '@/composables/config/useConnectionTestReport';
import { useMutation } from '@/composables/form/useMutation';
import { countLabel } from '@/lib/form/validation';

// How guests receive their photos by mail or through a cloud link, with a test of the cloud share
const { t } = useI18n();
const { state, fieldProps, savedSecrets, clearedSecrets, keepsSecret, hasSecret } = useConfigForm();
const { reportTestConnection } = useConnectionTestReport();

const { mutate: testShare, pending: testingShare } = useMutation(
  async () => configRepository.testShare(state.share.cloud),
  {
    onSuccess: reportTestConnection,
  },
);

const shareTestDisabled = computed(() => {
  const cloud = state.share.cloud;
  if (cloud.driver === 'nextcloud') {
    const nextcloud = cloud.nextcloud;
    return !(
      nextcloud.url.trim() &&
      nextcloud.username.trim() &&
      hasSecret('share.cloud.nextcloud.password', nextcloud.password)
    );
  }
  if (cloud.driver === 'self') {
    return !cloud.self.baseUrl.trim();
  }
  return false;
});

// The select reserves the empty string to clear the selection and rejects it as an item value.
// The option for no cloud driver therefore uses null and maps it to the empty driver of the configuration.
const cloudDriverSelection = computed<CloudShareDriver | null>({
  get: () => state.share.cloud.driver || null,
  set: (value) => {
    state.share.cloud.driver = value ?? '';
  },
});

const cloudDriverItems = computed((): { label: string; value: CloudShareDriver | null }[] => [
  {
    label: t('pages.config.share.cloudNone'),
    value: null,
  },
  {
    label: 'Nextcloud',
    value: 'nextcloud',
  },
  {
    label: t('pages.config.share.cloudSelf'),
    value: 'self',
  },
]);

// The configuration stores no driver as an empty value, which the select shows as null
const cloudDriverDefaults = computed(() =>
  cloudDriverItems.value.map((item) => ({
    label: item.label,
    value: item.value ?? '',
  })),
);
</script>

<template>
  <UCard variant="subtle">
    <div class="flex flex-col gap-4">
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <UFormField :label="t('fields.share.email.enabled')" name="share.email.enabled">
          <template #label="{ label }">
            <span class="inline-flex items-center gap-1.5">
              {{ label }}
              <ConfigFieldHint
                path="share.email.enabled"
                :help="t('pages.config.share.emailEnabledHint')"
              />
            </span>
          </template>
          <USwitch v-model="state.share.email.enabled" v-bind="fieldProps('share.email.enabled')" />
        </UFormField>
        <UFormField :label="t('fields.share.email.subject')" name="share.email.subject">
          <template #hint>
            <ConfigFieldHint
              path="share.email.subject"
              :count="countLabel(state.share.email.subject, CONFIG_LIMITS.shareEmailSubject)"
            />
          </template>
          <UInput
            v-model="state.share.email.subject"
            v-bind="fieldProps('share.email.subject')"
            :maxlength="CONFIG_LIMITS.shareEmailSubject"
            class="w-full"
          />
        </UFormField>
      </div>
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <UFormField :label="t('fields.share.cloud.driver')" name="share.cloud.driver">
          <template #hint>
            <ConfigFieldHint
              path="share.cloud.driver"
              :help="t('pages.config.share.cloudDriverHint')"
              :options="cloudDriverDefaults"
            />
          </template>
          <USelect
            v-model="cloudDriverSelection"
            v-bind="fieldProps('share.cloud.driver')"
            :items="cloudDriverItems"
            class="w-full"
          />
        </UFormField>
      </div>
      <div
        v-if="state.share.cloud.driver === 'nextcloud'"
        class="grid grid-cols-1 gap-4 sm:grid-cols-2"
      >
        <UFormField
          :label="t('fields.share.cloud.nextcloud.url')"
          name="share.cloud.nextcloud.url"
          required
        >
          <template #hint>
            <ConfigFieldHint path="share.cloud.nextcloud.url" />
          </template>
          <UInput
            v-model="state.share.cloud.nextcloud.url"
            v-bind="fieldProps('share.cloud.nextcloud.url')"
            class="w-full"
          />
        </UFormField>
        <UFormField
          :label="t('fields.share.cloud.nextcloud.username')"
          name="share.cloud.nextcloud.username"
          required
        >
          <template #hint>
            <ConfigFieldHint path="share.cloud.nextcloud.username" />
          </template>
          <UInput
            v-model="state.share.cloud.nextcloud.username"
            v-bind="fieldProps('share.cloud.nextcloud.username')"
            class="w-full"
          />
        </UFormField>
        <UFormField
          :label="t('fields.share.cloud.nextcloud.password')"
          name="share.cloud.nextcloud.password"
          :required="!keepsSecret('share.cloud.nextcloud.password')"
        >
          <template #hint>
            <ConfigFieldHint
              path="share.cloud.nextcloud.password"
              :help="t('pages.config.share.nextcloudPasswordHint')"
            />
          </template>
          <SecretField
            v-model="state.share.cloud.nextcloud.password"
            v-model:cleared="clearedSecrets['share.cloud.nextcloud.password']"
            :saved="!!savedSecrets['share.cloud.nextcloud.password']"
            v-bind="fieldProps('share.cloud.nextcloud.password')"
          />
        </UFormField>
        <UFormField
          :label="t('fields.share.cloud.nextcloud.folder')"
          name="share.cloud.nextcloud.folder"
        >
          <template #hint>
            <ConfigFieldHint
              path="share.cloud.nextcloud.folder"
              :help="t('pages.config.share.nextcloudFolderHint')"
              :count="
                countLabel(state.share.cloud.nextcloud.folder, CONFIG_LIMITS.shareNextcloudFolder)
              "
            />
          </template>
          <UInput
            v-model="state.share.cloud.nextcloud.folder"
            v-bind="fieldProps('share.cloud.nextcloud.folder')"
            :maxlength="CONFIG_LIMITS.shareNextcloudFolder"
            class="w-full"
          />
        </UFormField>
      </div>

      <div v-if="state.share.cloud.driver === 'self'" class="grid grid-cols-1 gap-4">
        <UFormField
          :label="t('fields.share.cloud.self.baseUrl')"
          name="share.cloud.self.baseUrl"
          required
        >
          <template #hint>
            <ConfigFieldHint
              path="share.cloud.self.baseUrl"
              :help="t('pages.config.share.cloudSelfBaseUrlHint')"
            />
          </template>
          <UInput
            v-model="state.share.cloud.self.baseUrl"
            v-bind="fieldProps('share.cloud.self.baseUrl')"
            placeholder="http://192.168.4.1:8080"
            class="w-full"
          />
        </UFormField>
      </div>

      <div class="flex items-center gap-1.5">
        <UButton
          :label="t('pages.config.testConnection')"
          icon="i-lucide-plug-zap"
          color="neutral"
          variant="outline"
          :loading="testingShare"
          :disabled="shareTestDisabled"
          @click="testShare()"
        />
      </div>
    </div>
  </UCard>
</template>
