<script setup lang="ts">
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { useMutation } from '@/composables/form/useMutation';

const props = defineProps<{
  title: string;
  description: string;
  actionLabel: string;
  icon: string;
  color: 'neutral' | 'warning' | 'error';
  confirmTitle: string;
  confirmText: string;
  successMessage: string;
  // The shell command the action runs, shown below the description.
  command?: string;
  disabled?: boolean;
  run: () => Promise<void>;
}>();

const { t } = useI18n();

const open = ref(false);

const { mutate, pending } = useMutation(() => props.run(), {
  success: props.successMessage,
  onSuccess: () => {
    open.value = false;
  },
});
</script>

<template>
  <UPageCard :title="title" :description="description" orientation="horizontal">
    <template v-if="command" #footer>
      <code class="text-muted bg-elevated rounded px-2 py-1 font-mono text-xs break-all">
        {{ command }}
      </code>
    </template>
    <UButton
      :label="actionLabel"
      :icon="icon"
      :color="color"
      :disabled="disabled"
      @click="open = true"
    />
  </UPageCard>

  <UModal
    v-model:open="open"
    :title="confirmTitle"
    :description="confirmText"
    :ui="{ footer: 'justify-end' }"
  >
    <template #footer="{ close }">
      <UButton :label="t('common.cancel')" color="neutral" variant="outline" @click="close" />
      <UButton :label="actionLabel" :color="color" :loading="pending" @click="mutate()" />
    </template>
  </UModal>
</template>
