<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import EditorPanel from '@/components/editor/EditorPanel.vue';
import ColorField from '@/components/form/ColorField.vue';
import { LAYOUT_LIMITS, useLayoutForm } from '@/composables/useLayoutForm';
import { countLabel } from '@/lib/form/validation';

// The caption printed on every page.
const { t } = useI18n();
const { state, fieldProps, brandingFontSizePt, brandingColor } = useLayoutForm();
</script>

<template>
  <EditorPanel
    :title="t('pages.layouts.panel.branding')"
    storage-key="layoutEditor.panel.branding"
    class="shrink-0"
  >
    <UFormField :label="t('fields.branding.enabled')" name="branding.enabled">
      <USwitch v-model="state.branding.enabled" v-bind="fieldProps('branding.enabled')" />
    </UFormField>
    <template v-if="state.branding.enabled">
      <UFormField
        :label="t('fields.branding.template')"
        :description="`${t('pages.layouts.brandingTemplateHint')} {{eventName}}, {{date}}`"
        :hint="countLabel(state.branding.template, LAYOUT_LIMITS.brandingTemplate)"
        name="branding.template"
      >
        <UInput
          v-model="state.branding.template"
          v-bind="fieldProps('branding.template')"
          :maxlength="LAYOUT_LIMITS.brandingTemplate"
          class="w-full"
        />
      </UFormField>
      <UFormField :label="t('fields.branding.rotationDeg')" name="branding.rotationDeg">
        <UInputNumber
          v-model="state.branding.rotationDeg"
          v-bind="fieldProps('branding.rotationDeg')"
          :min="-180"
          :max="180"
          class="w-full"
        />
      </UFormField>
      <div class="flex items-end gap-2">
        <UFormField
          :label="t('fields.branding.fontSizePt')"
          name="branding.fontSizePt"
          class="min-w-0 flex-1"
        >
          <UInputNumber v-model="brandingFontSizePt" :min="6" :max="72" class="w-full" />
        </UFormField>
        <UFormField :label="t('fields.branding.color')" name="branding.color">
          <ColorField v-model="brandingColor" />
        </UFormField>
      </div>
    </template>
  </EditorPanel>
</template>
