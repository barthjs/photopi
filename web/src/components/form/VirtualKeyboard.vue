<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  type KeyboardKey,
  type KeyboardLanguage,
  keyboardLayouts,
  symbolLayout,
  textLayouts,
} from '@/lib/keyboardLayouts';

const { modelValue, variant = 'email' } = defineProps<{
  modelValue: string;
  // email types an address, text types any text
  variant?: 'email' | 'text';
}>();
const emit = defineEmits<{ 'update:modelValue': [value: string] }>();

const { t, locale } = useI18n();

const language = computed<KeyboardLanguage>(() => (locale.value === 'de' ? 'de' : 'en'));
// The shift key applies to the next letter only, like on a phone
const shifted = ref(false);
const showSymbols = ref(false);

const layout = computed(() => {
  if (variant === 'email') return keyboardLayouts[language.value];
  return showSymbols.value ? symbolLayout : textLayouts[language.value];
});

/**
 * spanStyle returns the column span of a key as an inline style, since Tailwind only generates classes
 * that appear literally in the source.
 */
function spanStyle(span: number) {
  return {
    gridColumn: `span ${span} / span ${span}`,
  };
}

/**
 * charLabel returns what a character key types, the capital letter while shift is active.
 * A letter without a single capital letter, such as ß, stays as it is.
 */
function charLabel(label: string): string {
  if (!shifted.value) return label;
  const upper = label.toUpperCase();
  return upper.length === 1 ? upper : label;
}

/** keyLabel returns the text shown on a key, undefined for a key that only shows an icon. */
function keyLabel(key: KeyboardKey): string | undefined {
  switch (key.kind) {
    case 'char':
      return charLabel(key.label);
    case 'symbols':
      return showSymbols.value ? t('keyboard.letters') : t('keyboard.symbols');
    default:
      return undefined;
  }
}

/** keyIcon returns the icon of a key, undefined for a key that shows text. */
function keyIcon(key: KeyboardKey): string | undefined {
  switch (key.kind) {
    case 'backspace':
      return 'i-lucide-delete';
    case 'shift':
      return 'i-lucide-arrow-big-up';
    case 'space':
      return 'i-lucide-space';
    default:
      return undefined;
  }
}

/** keyVariant highlights the keys that do not type a character, and an active shift key. */
function keyVariant(key: KeyboardKey): 'solid' | 'outline' {
  if (key.kind === 'char') return 'outline';
  if (key.kind === 'shift') return shifted.value ? 'solid' : 'outline';
  return 'solid';
}

/** press types a key into the bound text, or runs the action of a key that types nothing. */
function press(key: KeyboardKey): void {
  switch (key.kind) {
    case 'spacer':
      return;
    case 'backspace':
      emit('update:modelValue', modelValue.slice(0, -1));
      return;
    case 'shift':
      shifted.value = !shifted.value;
      return;
    case 'symbols':
      showSymbols.value = !showSymbols.value;
      shifted.value = false;
      return;
    case 'space':
      emit('update:modelValue', `${modelValue} `);
      return;
    case 'char':
      emit('update:modelValue', modelValue + charLabel(key.label));
      shifted.value = false;
      return;
  }
}
</script>

<template>
  <div
    class="mx-auto grid aspect-3/1 w-full gap-1"
    :style="{ gridTemplateRows: `repeat(${layout.length}, minmax(0, 1fr))` }"
  >
    <div v-for="(row, rowIndex) in layout" :key="rowIndex" class="grid grid-cols-12 gap-1">
      <template v-for="(key, keyIndex) in row" :key="keyIndex">
        <div v-if="key.kind === 'spacer'" :style="spanStyle(key.span)" />
        <UButton
          v-else
          :label="keyLabel(key)"
          :icon="keyIcon(key)"
          color="neutral"
          :variant="keyVariant(key)"
          @click="press(key)"
          :style="spanStyle(key.span)"
          class="h-full w-full min-w-0 justify-center text-base"
        />
      </template>
    </div>
  </div>
</template>
