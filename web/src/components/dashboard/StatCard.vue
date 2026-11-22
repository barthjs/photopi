<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

// A value with its label and icon, which may lead to the page it stands for, inside the app or outside of it
const props = defineProps<{
  label: string;
  // A number is written in the format of the locale, a text as it is
  value: string | number;
  icon: string;
  // A page of the app the card leads to
  to?: string;
  // A page outside the app, which opens in a new tab
  href?: string;
  // Marks a number that asks for action while it is above zero
  highlight?: boolean;
  // Shows a longer text smaller and wrapped instead of cut off
  textValue?: boolean;
  // A further line below the label, such as the parts the value stands for
  detail?: string;
}>();

const { locale } = useI18n();

const formatted = computed(() =>
  typeof props.value === 'number'
    ? new Intl.NumberFormat(locale.value).format(props.value)
    : props.value,
);
const highlighted = computed(
  () => props.highlight && typeof props.value === 'number' && props.value > 0,
);
// A number that asks for action stays in the background while there is nothing to do
const idle = computed(() => props.highlight && props.value === 0);
const link = computed(() => props.to ?? props.href);
</script>

<template>
  <!-- Without a link the card renders in a plain element. A narrow card, such as one of three on a phone,
       puts the value beside a smaller icon and the label below both, and a wide card puts the icon
       beside the value and the label. -->
  <ULink
    :to="link"
    :target="href ? '_blank' : undefined"
    :rel="href ? 'noopener noreferrer' : undefined"
    as="div"
    class="group @container"
  >
    <UCard
      variant="subtle"
      class="h-full transition"
      :class="link && 'group-hover:ring-accented'"
      :ui="{
        body: 'grid grid-cols-[auto_minmax(0,1fr)] items-center gap-x-2 gap-y-1 p-3 sm:p-3 @[11rem]:gap-x-3 @[11rem]:gap-y-0 @[11rem]:p-4',
      }"
    >
      <div
        class="flex size-8 shrink-0 items-center justify-center rounded-lg @[11rem]:row-span-2 @[11rem]:size-10"
        :class="highlighted ? 'bg-info/10 text-info' : 'bg-elevated text-muted'"
      >
        <UIcon :name="icon" class="size-4 @[11rem]:size-5" />
      </div>
      <p
        class="min-w-0 font-semibold tabular-nums"
        :class="[
          idle ? 'text-muted' : 'text-highlighted',
          textValue ? 'text-lg wrap-break-word' : 'truncate text-2xl',
        ]"
        :title="formatted"
      >
        {{ formatted }}
      </p>
      <div class="col-span-2 min-w-0 @[11rem]:col-span-1 @[11rem]:col-start-2">
        <p class="text-muted truncate text-sm">{{ label }}</p>
        <p v-if="detail" class="text-toned mt-1 text-sm wrap-break-word">{{ detail }}</p>
      </div>
    </UCard>
  </ULink>
</template>
