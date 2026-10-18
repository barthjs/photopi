<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import ConfigFieldHint from '@/components/config/ConfigFieldHint.vue';
import SecretField from '@/components/form/SecretField.vue';
import { useConfigForm } from '@/composables/useConfigForm';

// The mail server that sends the shared photos.
const { t } = useI18n();
const { state, fieldProps, savedSecrets, clearedSecrets } = useConfigForm();

const driverItems = computed(() => [
  {
    label: t('pages.config.mail.driverLog'),
    value: 'log',
  },
  {
    label: 'SMTP',
    value: 'smtp',
  },
]);
</script>

<template>
  <UCard variant="subtle" class="mt-4">
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <UFormField :label="t('fields.mail.driver')" name="mail.driver" class="sm:col-span-2">
        <template #hint>
          <ConfigFieldHint
            path="mail.driver"
            :help="t('pages.config.mail.driverHint')"
            :options="driverItems"
          />
        </template>
        <USelect
          v-model="state.mail.driver"
          v-bind="fieldProps('mail.driver')"
          :items="driverItems"
          class="w-full sm:w-1/2"
        />
      </UFormField>
      <template v-if="state.mail.driver === 'smtp'">
        <UFormField :label="t('fields.mail.host')" name="mail.host" required>
          <template #hint>
            <ConfigFieldHint path="mail.host" />
          </template>
          <UInput v-model="state.mail.host" v-bind="fieldProps('mail.host')" class="w-full" />
        </UFormField>
        <UFormField :label="t('fields.mail.port')" name="mail.port">
          <template #hint>
            <ConfigFieldHint path="mail.port" />
          </template>
          <UInputNumber
            v-model="state.mail.port"
            v-bind="fieldProps('mail.port')"
            :min="1"
            :max="65535"
            class="w-full"
          />
        </UFormField>
        <UFormField :label="t('fields.mail.username')" name="mail.username">
          <template #hint>
            <ConfigFieldHint path="mail.username" />
          </template>
          <UInput
            v-model="state.mail.username"
            v-bind="fieldProps('mail.username')"
            class="w-full"
          />
        </UFormField>
        <UFormField :label="t('fields.mail.password')" name="mail.password">
          <template #hint>
            <ConfigFieldHint path="mail.password" />
          </template>
          <SecretField
            v-model="state.mail.password"
            v-model:cleared="clearedSecrets['mail.password']"
            :saved="!!savedSecrets['mail.password']"
            v-bind="fieldProps('mail.password')"
          />
        </UFormField>
        <UFormField :label="t('fields.mail.fromAddress')" name="mail.fromAddress">
          <template #hint>
            <ConfigFieldHint path="mail.fromAddress" />
          </template>
          <UInput
            v-model="state.mail.fromAddress"
            v-bind="fieldProps('mail.fromAddress')"
            type="email"
            class="w-full"
          />
        </UFormField>
        <UFormField :label="t('fields.mail.fromName')" name="mail.fromName">
          <template #hint>
            <ConfigFieldHint path="mail.fromName" />
          </template>
          <UInput
            v-model="state.mail.fromName"
            v-bind="fieldProps('mail.fromName')"
            class="w-full"
          />
        </UFormField>
        <UFormField :label="t('fields.mail.replyTo')" name="mail.replyTo">
          <template #hint>
            <ConfigFieldHint path="mail.replyTo" :help="t('pages.config.mail.replyToHint')" />
          </template>
          <UInput
            v-model="state.mail.replyTo"
            v-bind="fieldProps('mail.replyTo')"
            type="email"
            class="w-full"
          />
        </UFormField>
        <UFormField :label="t('fields.mail.messageIdDomain')" name="mail.messageIdDomain">
          <template #hint>
            <ConfigFieldHint
              path="mail.messageIdDomain"
              :help="t('pages.config.mail.messageIdDomainHint')"
            />
          </template>
          <UInput
            v-model="state.mail.messageIdDomain"
            v-bind="fieldProps('mail.messageIdDomain')"
            class="w-full"
          />
        </UFormField>
      </template>
    </div>
  </UCard>
</template>
