<script setup lang="ts">
import { computed, onMounted, ref, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import { useToast } from '@nuxt/ui/composables';

import { overlayRepository } from '@/api/repositories/overlay';
import type {
  CameraAutofocusMode,
  CameraExposureMode,
  CameraExposureTime,
  CameraDenoise,
  CameraRotation,
  CameraWhiteBalance,
  OverlayResponse,
} from '@/api/types';
import ConfigFieldHint from '@/components/config/ConfigFieldHint.vue';
import HelpHint from '@/components/form/HelpHint.vue';
import { useApiError } from '@/composables/useApiError';
import { useAspectRatioLock } from '@/composables/useAspectRatioLock';
import { useConfigForm } from '@/composables/useConfigForm';
import { useMutation } from '@/composables/form/useMutation';

// The camera that takes the photos and the overlay drawn over every capture.
const { t } = useI18n();
const toast = useToast();
const { messageFor } = useApiError();
const { state, fieldProps } = useConfigForm();

const {
  locked: captureRatioLocked,
  toggle: toggleCaptureRatioLock,
  label: captureRatioLabel,
} = useAspectRatioLock(
  {
    get: () => state.camera.captureWidth,
    set: (value) => (state.camera.captureWidth = value),
  },
  {
    get: () => state.camera.captureHeight,
    set: (value) => (state.camera.captureHeight = value),
  },
);
const {
  locked: previewRatioLocked,
  toggle: togglePreviewRatioLock,
  label: previewRatioLabel,
} = useAspectRatioLock(
  {
    get: () => state.camera.previewWidth,
    set: (value) => (state.camera.previewWidth = value),
  },
  {
    get: () => state.camera.previewHeight,
    set: (value) => (state.camera.previewHeight = value),
  },
);

const sourceItems = [
  {
    label: 'Webcam',
    value: 'webcam',
  },
  {
    label: 'rpicam',
    value: 'rpicam',
  },
  {
    label: 'gphoto2',
    value: 'gphoto2',
  },
];

// The exposure times a camera offers in its usual steps, from long to short.
const EXPOSURE_TIMES: CameraExposureTime[] = [
  '1/15',
  '1/20',
  '1/30',
  '1/40',
  '1/50',
  '1/60',
  '1/80',
  '1/100',
  '1/125',
  '1/160',
  '1/200',
  '1/250',
  '1/320',
  '1/400',
  '1/500',
];

const exposureTimeItems = computed<{ label: string; value: CameraExposureTime }[]>(() => [
  {
    label: t('pages.config.camera.exposureTimeAuto'),
    value: 'auto',
  },
  ...EXPOSURE_TIMES.map((time) => ({
    label: `${time} s`,
    value: time,
  })),
]);

const exposureModeItems = computed<{ label: string; value: CameraExposureMode }[]>(() => [
  {
    label: t('pages.config.camera.exposureModeNormal'),
    value: 'normal',
  },
  {
    label: t('pages.config.camera.exposureModeShort'),
    value: 'short',
  },
]);

const whiteBalanceItems = computed<{ label: string; value: CameraWhiteBalance }[]>(() => [
  {
    label: t('pages.config.camera.whiteBalanceAuto'),
    value: 'auto',
  },
  {
    label: t('pages.config.camera.whiteBalanceIncandescent'),
    value: 'incandescent',
  },
  {
    label: t('pages.config.camera.whiteBalanceTungsten'),
    value: 'tungsten',
  },
  {
    label: t('pages.config.camera.whiteBalanceFluorescent'),
    value: 'fluorescent',
  },
  {
    label: t('pages.config.camera.whiteBalanceIndoor'),
    value: 'indoor',
  },
  {
    label: t('pages.config.camera.whiteBalanceDaylight'),
    value: 'daylight',
  },
  {
    label: t('pages.config.camera.whiteBalanceCloudy'),
    value: 'cloudy',
  },
]);

const denoiseItems = computed<{ label: string; value: CameraDenoise }[]>(() => [
  {
    label: t('pages.config.camera.denoiseAuto'),
    value: 'auto',
  },
  {
    label: t('pages.config.camera.denoiseOff'),
    value: 'off',
  },
  {
    label: t('pages.config.camera.denoiseCdnOff'),
    value: 'cdn_off',
  },
  {
    label: t('pages.config.camera.denoiseCdnFast'),
    value: 'cdn_fast',
  },
  {
    label: t('pages.config.camera.denoiseCdnHq'),
    value: 'cdn_hq',
  },
]);

const autofocusModeItems = computed<{ label: string; value: CameraAutofocusMode }[]>(() => [
  {
    label: t('pages.config.camera.autofocusModeDefault'),
    value: 'default',
  },
  {
    label: t('pages.config.camera.autofocusModeManual'),
    value: 'manual',
  },
  {
    label: t('pages.config.camera.autofocusModeAuto'),
    value: 'auto',
  },
  {
    label: t('pages.config.camera.autofocusModeContinuous'),
    value: 'continuous',
  },
]);

const rotationItems: { label: string; value: CameraRotation }[] = [
  {
    label: '0°',
    value: 0,
  },
  {
    label: '180°',
    value: 180,
  },
];

const overlays = ref<OverlayResponse[]>([]);
const fileInput = useTemplateRef<HTMLInputElement>('fileInput');

// A configured overlay that is not listed, such as one set in the environment, shows the name in its path
const selectedOverlayName = computed(
  () =>
    overlays.value.find((overlay) => overlay.path === state.camera.overlayPath)?.name ??
    state.camera.overlayPath.split('/').pop(),
);

/** clearOverlay selects no overlay. */
function clearOverlay(): void {
  state.camera.overlayPath = '';
}

/** selectOverlay selects the overlay stored at the given path. */
function selectOverlay(path: string): void {
  state.camera.overlayPath = path;
}

/** loadOverlays reads the uploaded overlays. */
async function loadOverlays(): Promise<void> {
  try {
    overlays.value = await overlayRepository.list();
  } catch (err) {
    toast.add({
      title: messageFor(err),
      icon: 'i-lucide-circle-alert',
      color: 'error',
    });
  }
}

onMounted(loadOverlays);

const { mutate: uploadOverlay, pending: uploading } = useMutation(
  (file: File) => overlayRepository.create(file),
  {
    onSuccess: async (overlay) => {
      await loadOverlays();
      state.camera.overlayPath = overlay.path;
    },
  },
);

const { mutate: deleteOverlay } = useMutation((name: string) => overlayRepository.delete(name), {
  onSuccess: async (_result, name) => {
    if (state.camera.overlayPath.endsWith(`/${name}`)) {
      state.camera.overlayPath = '';
    }
    await loadOverlays();
  },
});

/** onOverlayFileChange uploads the chosen file and resets the input, so the same file can be chosen again. */
function onOverlayFileChange(event: Event): void {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = '';
  if (file) void uploadOverlay(file);
}
</script>

<template>
  <UCard variant="subtle" class="mt-4">
    <div class="flex flex-col gap-4">
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <UFormField :label="t('fields.camera.source')" name="camera.source">
          <template #hint>
            <ConfigFieldHint
              path="camera.source"
              :help="t('pages.config.camera.sourceHint')"
              :options="sourceItems"
            />
          </template>
          <USelect
            v-model="state.camera.source"
            v-bind="fieldProps('camera.source')"
            :items="sourceItems"
            class="w-full"
          />
        </UFormField>
      </div>

      <div v-if="state.camera.source === 'rpicam'" class="flex flex-col gap-4">
        <div class="grid grid-cols-2 gap-4">
          <UFormField :label="t('fields.camera.captureWidth')" name="camera.captureWidth" required>
            <template #hint>
              <ConfigFieldHint path="camera.captureWidth" />
            </template>
            <UInputNumber
              v-model="state.camera.captureWidth"
              v-bind="fieldProps('camera.captureWidth')"
              :min="1"
              :max="10000"
              class="w-full"
            />
          </UFormField>
          <UFormField
            :label="t('fields.camera.captureHeight')"
            name="camera.captureHeight"
            required
          >
            <template #hint>
              <ConfigFieldHint path="camera.captureHeight" />
            </template>
            <UInputNumber
              v-model="state.camera.captureHeight"
              v-bind="fieldProps('camera.captureHeight')"
              :min="1"
              :max="10000"
              class="w-full"
            />
          </UFormField>
          <div class="col-span-2 -mt-2 flex items-center gap-2">
            <UButton
              :icon="captureRatioLocked ? 'i-lucide-lock' : 'i-lucide-lock-open'"
              :color="captureRatioLocked ? 'primary' : 'neutral'"
              :variant="captureRatioLocked ? 'solid' : 'outline'"
              size="xs"
              :aria-label="t('pages.config.camera.aspectRatioLock')"
              @click="toggleCaptureRatioLock"
            />
            <span class="text-muted text-sm">
              {{ t('pages.config.camera.aspectRatioLabel', { ratio: captureRatioLabel }) }}
            </span>
            <HelpHint :text="t('pages.config.camera.aspectRatioHint')" />
          </div>
          <UFormField :label="t('fields.camera.previewWidth')" name="camera.previewWidth" required>
            <template #hint>
              <ConfigFieldHint path="camera.previewWidth" />
            </template>
            <UInputNumber
              v-model="state.camera.previewWidth"
              v-bind="fieldProps('camera.previewWidth')"
              :min="1"
              :max="4000"
              class="w-full"
            />
          </UFormField>
          <UFormField
            :label="t('fields.camera.previewHeight')"
            name="camera.previewHeight"
            required
          >
            <template #hint>
              <ConfigFieldHint path="camera.previewHeight" />
            </template>
            <UInputNumber
              v-model="state.camera.previewHeight"
              v-bind="fieldProps('camera.previewHeight')"
              :min="1"
              :max="4000"
              class="w-full"
            />
          </UFormField>
          <div class="col-span-2 -mt-2 flex items-center gap-2">
            <UButton
              :icon="previewRatioLocked ? 'i-lucide-lock' : 'i-lucide-lock-open'"
              :color="previewRatioLocked ? 'primary' : 'neutral'"
              :variant="previewRatioLocked ? 'solid' : 'outline'"
              size="xs"
              :aria-label="t('pages.config.camera.aspectRatioLock')"
              @click="togglePreviewRatioLock"
            />
            <span class="text-muted text-sm">
              {{ t('pages.config.camera.aspectRatioLabel', { ratio: previewRatioLabel }) }}
            </span>
          </div>
        </div>
        <UFormField
          :label="t('fields.camera.previewFps')"
          name="camera.previewFps"
          required
          class="sm:max-w-[calc(50%-0.5rem)]"
        >
          <template #hint>
            <ConfigFieldHint
              path="camera.previewFps"
              :help="t('pages.config.camera.previewFpsHint')"
            />
          </template>
          <UInputNumber
            v-model="state.camera.previewFps"
            v-bind="fieldProps('camera.previewFps')"
            :min="1"
            :max="120"
            class="w-full"
          />
        </UFormField>

        <USeparator :label="t('pages.config.camera.imageSettings')" />

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <UFormField :label="t('fields.camera.exposureTime')" name="camera.exposureTime">
            <template #hint>
              <ConfigFieldHint
                path="camera.exposureTime"
                :help="t('pages.config.camera.exposureTimeHint')"
                :options="exposureTimeItems"
              />
            </template>
            <USelect
              v-model="state.camera.exposureTime"
              v-bind="fieldProps('camera.exposureTime')"
              :items="exposureTimeItems"
              class="w-full"
            />
          </UFormField>
          <UFormField :label="t('fields.camera.exposureMode')" name="camera.exposureMode">
            <template #hint>
              <ConfigFieldHint
                path="camera.exposureMode"
                :help="t('pages.config.camera.exposureModeHint')"
                :options="exposureModeItems"
              />
            </template>
            <USelect
              v-model="state.camera.exposureMode"
              v-bind="fieldProps('camera.exposureMode')"
              :items="exposureModeItems"
              :disabled="state.camera.exposureTime !== 'auto'"
              class="w-full"
            />
          </UFormField>
          <UFormField :label="t('fields.camera.analogueGain')" name="camera.analogueGain">
            <template #hint>
              <ConfigFieldHint
                path="camera.analogueGain"
                :help="t('pages.config.camera.analogueGainHint')"
              />
            </template>
            <UInputNumber
              v-model="state.camera.analogueGain"
              v-bind="fieldProps('camera.analogueGain')"
              :min="0"
              :max="32"
              :step="0.5"
              class="w-full"
            />
          </UFormField>
          <UFormField
            :label="t('fields.camera.exposureCompensation')"
            name="camera.exposureCompensation"
          >
            <template #hint>
              <ConfigFieldHint
                path="camera.exposureCompensation"
                :help="t('pages.config.camera.exposureCompensationHint')"
              />
            </template>
            <UInputNumber
              v-model="state.camera.exposureCompensation"
              v-bind="fieldProps('camera.exposureCompensation')"
              :min="-10"
              :max="10"
              :step="0.5"
              class="w-full"
            />
          </UFormField>
          <UFormField :label="t('fields.camera.whiteBalance')" name="camera.whiteBalance">
            <template #hint>
              <ConfigFieldHint
                path="camera.whiteBalance"
                :help="t('pages.config.camera.whiteBalanceHint')"
                :options="whiteBalanceItems"
              />
            </template>
            <USelect
              v-model="state.camera.whiteBalance"
              v-bind="fieldProps('camera.whiteBalance')"
              :items="whiteBalanceItems"
              class="w-full"
            />
          </UFormField>
          <UFormField :label="t('fields.camera.denoise')" name="camera.denoise">
            <template #hint>
              <ConfigFieldHint
                path="camera.denoise"
                :help="t('pages.config.camera.denoiseHint')"
                :options="denoiseItems"
              />
            </template>
            <USelect
              v-model="state.camera.denoise"
              v-bind="fieldProps('camera.denoise')"
              :items="denoiseItems"
              class="w-full"
            />
          </UFormField>
          <UFormField :label="t('fields.camera.autofocusMode')" name="camera.autofocusMode">
            <template #hint>
              <ConfigFieldHint
                path="camera.autofocusMode"
                :help="t('pages.config.camera.autofocusModeHint')"
                :options="autofocusModeItems"
              />
            </template>
            <USelect
              v-model="state.camera.autofocusMode"
              v-bind="fieldProps('camera.autofocusMode')"
              :items="autofocusModeItems"
              class="w-full"
            />
          </UFormField>
          <UFormField :label="t('fields.camera.jpegQuality')" name="camera.jpegQuality" required>
            <template #hint>
              <ConfigFieldHint
                path="camera.jpegQuality"
                :help="t('pages.config.camera.jpegQualityHint')"
              />
            </template>
            <UInputNumber
              v-model="state.camera.jpegQuality"
              v-bind="fieldProps('camera.jpegQuality')"
              :min="1"
              :max="100"
              class="w-full"
            />
          </UFormField>
          <UFormField :label="t('fields.camera.rotation')" name="camera.rotation">
            <template #hint>
              <ConfigFieldHint
                path="camera.rotation"
                :help="t('pages.config.camera.rotationHint')"
                :options="rotationItems"
              />
            </template>
            <USelect
              v-model="state.camera.rotation"
              v-bind="fieldProps('camera.rotation')"
              :items="rotationItems"
              class="w-full"
            />
          </UFormField>
          <UFormField :label="t('fields.camera.horizontalFlip')" name="camera.horizontalFlip">
            <template #hint>
              <ConfigFieldHint
                path="camera.horizontalFlip"
                :help="t('pages.config.camera.flipHint')"
              />
            </template>
            <USwitch
              v-model="state.camera.horizontalFlip"
              v-bind="fieldProps('camera.horizontalFlip')"
            />
          </UFormField>
          <UFormField :label="t('fields.camera.verticalFlip')" name="camera.verticalFlip">
            <template #hint>
              <ConfigFieldHint
                path="camera.verticalFlip"
                :help="t('pages.config.camera.flipHint')"
              />
            </template>
            <USwitch
              v-model="state.camera.verticalFlip"
              v-bind="fieldProps('camera.verticalFlip')"
            />
          </UFormField>
        </div>
      </div>

      <UFormField :label="t('fields.camera.overlayPath')" name="camera.overlayPath">
        <template #hint>
          <ConfigFieldHint
            path="camera.overlayPath"
            :help="t('pages.config.camera.overlayPathHint')"
          />
        </template>
        <USelectMenu
          v-model="state.camera.overlayPath"
          v-bind="fieldProps('camera.overlayPath')"
          value-key="path"
          label-key="name"
          :items="overlays"
          :placeholder="t('pages.config.camera.overlayNone')"
          class="w-full"
        >
          <template #leading>
            <UButton
              v-if="state.camera.overlayPath"
              icon="i-lucide-x"
              size="xs"
              color="neutral"
              variant="ghost"
              :aria-label="t('pages.config.camera.overlayClear')"
              @click.stop="clearOverlay"
            />
          </template>
          <template #default="{ ui }">
            <span
              v-if="selectedOverlayName"
              data-slot="value"
              :class="ui.value?.({ class: undefined })"
            >
              {{ selectedOverlayName }}
            </span>
            <span v-else data-slot="placeholder" :class="ui.placeholder?.({ class: undefined })">
              {{ t('pages.config.camera.overlayNone') }}
            </span>
          </template>
        </USelectMenu>
      </UFormField>

      <UCard :ui="{ body: 'flex flex-col gap-3' }">
        <template #header>
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-1.5">
              <span class="text-highlighted font-medium">
                {{ t('pages.config.camera.overlayLibrary') }}
              </span>
              <HelpHint :text="t('pages.config.camera.overlayLibraryHint')" />
            </div>
            <UButton
              :label="t('pages.config.camera.uploadOverlay')"
              icon="i-lucide-upload"
              size="sm"
              :loading="uploading"
              @click="fileInput?.click()"
            />
            <input
              ref="fileInput"
              type="file"
              accept="image/png"
              class="hidden"
              @change="onOverlayFileChange"
            />
          </div>
        </template>

        <p v-if="!overlays.length" class="text-muted text-sm">
          {{ t('pages.config.camera.overlayEmpty') }}
        </p>
        <div
          v-for="overlay in overlays"
          :key="overlay.path"
          class="border-default flex items-center justify-between gap-2 rounded-md border px-3 py-2"
        >
          <span class="truncate text-sm">{{ overlay.name }}</span>
          <div class="flex shrink-0 items-center gap-2">
            <UBadge
              v-if="state.camera.overlayPath === overlay.path"
              color="primary"
              variant="solid"
            >
              {{ t('pages.config.camera.overlayActive') }}
            </UBadge>
            <UButton
              v-else
              :label="t('pages.config.camera.overlaySelect')"
              size="xs"
              color="neutral"
              variant="outline"
              @click="selectOverlay(overlay.path)"
            />
            <UButton
              icon="i-lucide-download"
              size="xs"
              color="neutral"
              variant="ghost"
              :to="overlayRepository.downloadUrl(overlay.name)"
              target="_blank"
              :aria-label="t('pages.config.camera.overlayDownload')"
            />
            <UButton
              icon="i-lucide-trash-2"
              size="xs"
              color="error"
              variant="ghost"
              :aria-label="t('pages.config.camera.overlayDelete')"
              @click="deleteOverlay(overlay.name)"
            />
          </div>
        </div>
      </UCard>
    </div>
  </UCard>
</template>
