<script setup lang="ts">
import { computed, ref, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import { useToast } from '@nuxt/ui/composables';
import { useIntersectionObserver } from '@vueuse/core';

import { layoutRepository } from '@/api/repositories/layout';
import { printerRepository } from '@/api/repositories/printer';
import type { PrintSubmissionMode } from '@/api/types/config';
import type { LayoutResponse } from '@/api/types/layout';
import type { PrinterResponse } from '@/api/types/printer';
import ConfigFieldHint from '@/components/config/ConfigFieldHint.vue';
import { useApiError } from '@/composables/form/useApiError';
import { useConfigForm } from '@/composables/config/useConfigForm';
import { useConfirm } from '@/composables/ui/useConfirm';
import { useConnectionTestReport } from '@/composables/config/useConnectionTestReport';
import { useMutation } from '@/composables/form/useMutation';
import { layoutMenuItems } from '@/lib/layout/layoutItems';

// The printer and the layout used to print a run, with a test of the printer
const { t } = useI18n();
const toast = useToast();
const { messageFor } = useApiError();
const { state, fieldProps } = useConfigForm();
const { reportTestConnection } = useConnectionTestReport();
const { confirm } = useConfirm();

const submissionModeItems = computed((): { label: string; value: PrintSubmissionMode }[] => [
  {
    label: t('pages.config.print.submissionModeManaged'),
    value: 'managed',
  },
  {
    label: t('pages.config.print.submissionModeRaw'),
    value: 'raw',
  },
]);

const printers = ref<PrinterResponse[]>([]);
const layouts = ref<LayoutResponse[]>([]);
const loadingPrinters = ref(false);
const loadingLayouts = ref(false);

/** Reads the printers the print system knows. */
async function loadPrinters(): Promise<void> {
  loadingPrinters.value = true;
  try {
    printers.value = await printerRepository.list();
  } catch (err) {
    toast.add({
      title: messageFor(err),
      icon: 'i-lucide-circle-alert',
      color: 'error',
    });
  } finally {
    loadingPrinters.value = false;
  }
}

/** Reads the stored print layouts. */
async function loadLayouts(): Promise<void> {
  loadingLayouts.value = true;
  try {
    layouts.value = await layoutRepository.list();
  } catch (err) {
    toast.add({
      title: messageFor(err),
      icon: 'i-lucide-circle-alert',
      color: 'error',
    });
  } finally {
    loadingLayouts.value = false;
  }
}

// The configuration page shows every section at once, so the print system is only asked once the section is in view
const card = useTemplateRef('card');
const { stop: stopObserving } = useIntersectionObserver(card, ([entry]) => {
  if (!entry?.isIntersecting) return;
  stopObserving();
  void loadPrinters();
  void loadLayouts();
});

const selectedPrinterStatus = computed(
  () => printers.value.find((printer) => printer.name === state.print.printerName)?.status,
);

const selectedLayout = computed(() =>
  layouts.value.find((layout) => layout.name === state.print.layout),
);

// A layout with another number of slots than photos per run leaves slots blank or photos unprinted
const slotMismatch = computed(
  () =>
    selectedLayout.value !== undefined && selectedLayout.value.slots !== state.capture.imagesPerRun,
);

const { mutate: resumePrinter, pending: resumingPrinter } = useMutation(
  async () =>
    printerRepository.resume({
      printerName: state.print.printerName,
    }),
  {
    success: () => t('pages.config.print.resumeSuccess'),
    onSuccess: loadPrinters,
  },
);

const { mutate: cancelJobs, pending: cancelingJobs } = useMutation(
  async () =>
    printerRepository.cancel({
      printerName: state.print.printerName,
    }),
  {
    confirm: async () =>
      confirm({
        title: t('pages.config.print.cancelConfirmTitle'),
        description: t('pages.config.print.cancelConfirmText', {
          printer: state.print.printerName,
        }),
        confirmLabel: t('pages.config.print.cancelJobs'),
        color: 'error',
        icon: 'i-lucide-list-x',
      }),
    success: () => t('pages.config.print.cancelSuccess'),
    onSuccess: loadPrinters,
  },
);

const { mutate: testPrinter, pending: testingPrinter } = useMutation(
  async () =>
    printerRepository.test({
      printerName: state.print.printerName,
    }),
  {
    onSuccess: reportTestConnection,
  },
);
</script>

<template>
  <UCard ref="card" variant="subtle">
    <div class="flex flex-col gap-4">
      <p class="text-muted text-sm">{{ t('pages.config.print.hint') }}</p>

      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <UFormField :label="t('fields.print.enabled')" name="print.enabled">
          <template #label="{ label }">
            <span class="inline-flex items-center gap-1.5">
              {{ label }}
              <ConfigFieldHint path="print.enabled" />
            </span>
          </template>
          <USwitch v-model="state.print.enabled" v-bind="fieldProps('print.enabled')" />
        </UFormField>
      </div>

      <div v-if="state.print.enabled" class="grid grid-cols-1 items-end gap-4 sm:grid-cols-2">
        <UFormField
          :label="t('fields.print.printerName')"
          :description="
            selectedPrinterStatus &&
            t('pages.config.print.printerStatus', { status: selectedPrinterStatus })
          "
          name="print.printerName"
          required
        >
          <template #hint>
            <ConfigFieldHint path="print.printerName" />
          </template>
          <USelectMenu
            v-model="state.print.printerName"
            v-bind="fieldProps('print.printerName')"
            value-key="name"
            label-key="name"
            :items="printers"
            :placeholder="t('pages.config.print.printerNone')"
            class="w-full"
          >
            <template #trailing>
              <UButton
                icon="i-lucide-refresh-cw"
                size="xs"
                color="neutral"
                variant="ghost"
                :loading="loadingPrinters"
                :aria-label="t('pages.config.print.reloadPrinters')"
                @click.stop="loadPrinters"
              />
            </template>
          </USelectMenu>
        </UFormField>

        <UFormField
          :label="t('fields.print.layout')"
          :description="
            selectedLayout &&
            t('pages.config.print.layoutDescription', {
              width: selectedLayout.paperWidthMM,
              height: selectedLayout.paperHeightMM,
              slots: selectedLayout.slots,
            })
          "
          name="print.layout"
          required
        >
          <template #hint>
            <ConfigFieldHint path="print.layout" :help="t('pages.config.print.layoutHint')" />
          </template>
          <USelectMenu
            v-model="state.print.layout"
            v-bind="fieldProps('print.layout')"
            value-key="name"
            label-key="name"
            :items="layoutMenuItems(layouts)"
            :placeholder="t('pages.config.print.layoutNone')"
            class="w-full"
          >
            <template #trailing>
              <UButton
                icon="i-lucide-refresh-cw"
                size="xs"
                color="neutral"
                variant="ghost"
                :loading="loadingLayouts"
                :aria-label="t('pages.config.print.reloadLayouts')"
                @click.stop="loadLayouts"
              />
            </template>
          </USelectMenu>
        </UFormField>
        <UFormField :label="t('fields.print.submissionMode')" name="print.submissionMode">
          <template #hint>
            <ConfigFieldHint
              path="print.submissionMode"
              :help="t('pages.config.print.submissionModeHint')"
              :options="submissionModeItems"
            />
          </template>
          <USelect
            v-model="state.print.submissionMode"
            v-bind="fieldProps('print.submissionMode')"
            :items="submissionModeItems"
            class="w-full"
          />
        </UFormField>
      </div>

      <UAlert
        v-if="state.print.enabled && slotMismatch && selectedLayout"
        color="warning"
        variant="subtle"
        icon="i-lucide-triangle-alert"
        :title="
          t('pages.config.print.slotMismatch', {
            slots: selectedLayout.slots,
            images: state.capture.imagesPerRun,
          })
        "
      />

      <div>
        <UButton
          :label="t('pages.config.print.manageLayouts')"
          icon="i-lucide-layout-grid"
          size="sm"
          color="neutral"
          variant="link"
          :to="{ name: 'layouts' }"
        />
      </div>

      <div v-if="state.print.enabled" class="flex flex-wrap items-center gap-1.5">
        <UButton
          :label="t('pages.config.testConnection')"
          icon="i-lucide-plug-zap"
          color="neutral"
          variant="outline"
          :loading="testingPrinter"
          :disabled="!state.print.printerName"
          @click="testPrinter()"
        />
        <UButton
          :label="t('pages.config.print.resume')"
          icon="i-lucide-play"
          color="neutral"
          variant="outline"
          class="ms-2"
          :loading="resumingPrinter"
          :disabled="!state.print.printerName"
          @click="resumePrinter()"
        />
        <UButton
          :label="t('pages.config.print.cancelJobs')"
          icon="i-lucide-list-x"
          color="error"
          variant="outline"
          class="ms-2"
          :loading="cancelingJobs"
          :disabled="!state.print.printerName"
          @click="cancelJobs()"
        />
      </div>
    </div>
  </UCard>
</template>
