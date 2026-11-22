<script setup lang="ts">
import { useMutation } from '@/composables/form/useMutation';
import { useConfirm } from '@/composables/ui/useConfirm';

// A button that runs a system action after a confirmation and reports that it was triggered
const props = defineProps<{
  label: string;
  icon: string;
  color: 'neutral' | 'warning' | 'error';
  confirmTitle: string;
  confirmText: string;
  successMessage: string;
  disabled?: boolean;
  run: () => Promise<void>;
}>();

const { confirm } = useConfirm();

const { mutate, pending } = useMutation(async () => props.run(), {
  confirm: async () =>
    confirm({
      title: props.confirmTitle,
      description: props.confirmText,
      confirmLabel: props.label,
      color: props.color,
      icon: props.icon,
    }),
  success: props.successMessage,
});
</script>

<template>
  <UButton
    :label="label"
    :icon="icon"
    :color="color"
    variant="subtle"
    :disabled="disabled"
    :loading="pending"
    @click="mutate()"
  />
</template>
