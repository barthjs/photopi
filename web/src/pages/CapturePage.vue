<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { cameraRepository } from '@/api/repositories/camera';
import { captureRepository } from '@/api/repositories/capture';

import { useApiError } from '@/composables/form/useApiError';
import { useCountdown } from '@/composables/timer/useCountdown';
import { useInactivityTimer } from '@/composables/timer/useInactivityTimer';

import { useBoothStore } from '@/stores/booth';
import { useMetaStore } from '@/stores/meta';

const { t } = useI18n();
const router = useRouter();

const PRE_START_TIMEOUT_MS = 60_000;
// How long the next countdown waits for the preview stream to reconnect after a shot.
// The sequence goes on without it, since the capture does not need the preview.
const PREVIEW_READY_TIMEOUT_MS = 5_000;
// How long a broken preview stream waits before it connects again, so a camera that keeps failing is not hammered
const PREVIEW_RETRY_DELAY_MS = 2_000;
const COUNTDOWN_SECONDS = 10;
// How long the webcam gets to adjust its exposure to the white screen before its frame is grabbed.
// The browser offers no way to learn when it settled, so this is a fixed allowance.
const WEBCAM_LIGHT_SETTLE_MS = 500;
const CAPTURE_ERROR_COUNTDOWN_SECONDS = 10;
// The size a webcam is asked for, so the browser picks the largest mode up to 4K
const WEBCAM_IDEAL_WIDTH = 3840;
const WEBCAM_IDEAL_HEIGHT = 2160;
// How long the photo just taken is shown before the next countdown, so the guests see that it worked.
// The live view connects again meanwhile.
const SHOT_DISPLAY_MS = 3_000;

const { messageFor } = useApiError();
const countdown = useCountdown();

const countdownPercent = computed(() => {
  const total = countdown.total.value;
  const remaining = countdown.remaining.value;
  return total > 0 ? Math.min(100, Math.round(((total - remaining) / total) * 100)) : 0;
});

const countdownRingStyle = computed(() => ({
  background: `conic-gradient(var(--ui-color-primary-500) ${countdownPercent.value}%, color-mix(in oklab, var(--ui-color-primary-500) 18%, transparent) 0)`,
}));

const booth = useBoothStore();
const metaStore = useMetaStore();
const source = computed(() => metaStore.meta?.camera.source ?? 'webcam');
const hasOverlay = computed(() => metaStore.meta?.camera.hasOverlay ?? false);
const upcomingIndex = computed(() => Math.min(booth.capturedCount + 1, booth.imagesPerRun ?? 0));

type Phase = 'idle' | 'counting' | 'error';
const phase = ref<Phase>('idle');
// True from the end of a countdown until the photo is taken. The screen turns white and lights the guests,
// which in a dark booth is the only light of the photo, so it stays white during the whole exposure.
// A backend camera needs a moment to switch from the live view to the photo,
// and the white screen also asks the guests to hold still until then and covers the stopped live view.
const lighting = ref(false);
// The image index of the photo just taken while it is shown, null otherwise
const shownShot = ref<number | null>(null);
// True from the start of a countdown until its light,
// so the ring stays hidden while waitForPreviewReady waits for the backend.
const ringVisible = ref(false);
const errorMessage = ref('');
// Only the webcam source needs the browser to open the camera.
// The backend drives every other source, so they start out ready.
// eslint-disable-next-line vue/no-ref-object-reactivity-loss -- set once from the source.
const cameraReady = ref(source.value !== 'webcam');

// A source driven by the backend holds one connection to the camera at a time, so a shot stops the live view.
// Increasing this makes the image open a new connection for the next shot of the run.
const previewReloadKey = ref(0);
// Whether the preview stream delivered its first frame, so a loading overlay covers the gap while the connection opens
const previewLoaded = ref(false);
watch(previewReloadKey, () => {
  previewLoaded.value = false;
});

let previewRetryTimer: ReturnType<typeof setTimeout> | null = null;

/** Connects the preview stream again after a delay, unless a photo in progress reconnects it afterward. */
function onPreviewError(): void {
  if (lighting.value || previewRetryTimer !== null) return;

  previewRetryTimer = setTimeout(() => {
    previewRetryTimer = null;
    if (!unmounted && !lighting.value) previewReloadKey.value++;
  }, PREVIEW_RETRY_DELAY_MS);
}

const videoEl = useTemplateRef<HTMLVideoElement>('videoEl');
const previewImgEl = useTemplateRef<HTMLImageElement>('previewImgEl');
let mediaStream: MediaStream | null = null;
// Set on unmount, so work that finishes later, such as opening the webcam, releases what it got
let unmounted = false;

// The full-screen box the media and its overlay fit into without cropping.
// It is measured once the ratio of the media is known, since the kiosk window does not resize.
const previewContainer = useTemplateRef<HTMLElement>('previewContainer');
// The width to height ratio of the delivered frames, since not every source has a configured resolution
const mediaAspectRatio = ref<number | null>(null);
// The largest rectangle of mediaAspectRatio inside previewContainer. It is computed by hand
// instead of with object-fit, since the overlay has to fill the same rectangle as the media.
const mediaBoxStyle = ref<{ width: string; height: string }>({
  width: '100%',
  height: '100%',
});

/** Sizes the media box to the largest rectangle of the given ratio that fits the preview container. */
function updateMediaBox(ratio: number): void {
  mediaAspectRatio.value = ratio;

  const container = previewContainer.value;
  if (!container) return;
  const { width: containerWidth, height: containerHeight } = container.getBoundingClientRect();
  if (!containerWidth || !containerHeight) return;

  mediaBoxStyle.value =
    ratio > containerWidth / containerHeight
      ? {
          width: '100%',
          height: `${containerWidth / ratio}px`,
        }
      : {
          width: `${containerHeight * ratio}px`,
          height: '100%',
        };
}

/** Sizes the media box once the webcam reports its resolution. */
function onVideoMetadata(): void {
  const video = videoEl.value;
  if (video && video.videoWidth > 0 && video.videoHeight > 0) {
    updateMediaBox(video.videoWidth / video.videoHeight);
  }
}

/** Marks the preview as loaded and sizes the media box to its frame. */
function onPreviewLoad(event: Event): void {
  previewLoaded.value = true;

  const img = event.target;
  if (img instanceof HTMLImageElement && img.naturalWidth > 0 && img.naturalHeight > 0) {
    updateMediaBox(img.naturalWidth / img.naturalHeight);
  }
}

const inactivity = useInactivityTimer(() => {
  void booth.abort().finally(async () =>
    router.push({
      name: 'welcome',
    }),
  );
});

onMounted(async () => {
  if (booth.run === null || booth.imagesPerRun === null) {
    await router.replace({
      name: 'welcome',
    });
    return;
  }

  inactivity.arm(PRE_START_TIMEOUT_MS);

  if (source.value === 'webcam') {
    try {
      // Without constraints, browsers often pick a low resolution.
      // An ideal instead of an exact size lets the browser choose the largest mode up to it
      // without failing on cameras that lack it.
      const stream = await navigator.mediaDevices.getUserMedia({
        video: {
          width: {
            ideal: WEBCAM_IDEAL_WIDTH,
          },
          height: {
            ideal: WEBCAM_IDEAL_HEIGHT,
          },
        },
      });
      // The page may be gone by the time the browser granted the camera, which must not stay on then
      if (unmounted) {
        stream.getTracks().forEach((track) => track.stop());
        return;
      }
      mediaStream = stream;
      if (videoEl.value) videoEl.value.srcObject = mediaStream;
      cameraReady.value = true;
    } catch (err) {
      await failAndReturnToWelcome(err);
    }
  }
});

onBeforeUnmount(() => {
  unmounted = true;
  if (previewRetryTimer !== null) clearTimeout(previewRetryTimer);
  mediaStream?.getTracks().forEach((track) => track.stop());

  // Removing the image does not reliably close a multipart stream, which would keep the camera
  // and a connection of the browser busy. Clearing the source aborts the request.
  if (previewImgEl.value) previewImgEl.value.src = '';
});

/**
 * Waits until the preview stream delivered its first frame after a shot,
 * so the next countdown does not start while the backend switches back to the live view.
 * The webcam stream never stops, so it does not wait for it.
 */
async function waitForPreviewReady(): Promise<void> {
  if (source.value === 'webcam' || previewLoaded.value) return Promise.resolve();

  return new Promise((resolve) => {
    let settled = false;

    const stopWatch = watch(previewLoaded, (loaded) => {
      if (!loaded || settled) return;
      settled = true;
      stopWatch();
      clearTimeout(timeoutId);
      resolve();
    });

    const timeoutId = setTimeout(() => {
      if (settled) return;
      settled = true;
      stopWatch();
      resolve();
    }, PREVIEW_READY_TIMEOUT_MS);
  });
}

/**
 * Returns the current webcam frame as a JPEG. Every other source takes its photo on the backend.
 */
async function grabFrame(): Promise<Blob> {
  const video = videoEl.value;
  if (!video) throw new Error('camera preview not ready');

  const canvas = document.createElement('canvas');
  canvas.width = video.videoWidth;
  canvas.height = video.videoHeight;
  canvas.getContext('2d')?.drawImage(video, 0, 0);

  return new Promise<Blob>((resolve, reject) => {
    canvas.toBlob(
      (blob) => (blob ? resolve(blob) : reject(new Error('failed to encode frame'))),
      'image/jpeg',
      0.92,
    );
  });
}

/**
 * Takes one photo of the run while the screen lights the guests.
 * The light stays on until the photo is taken, and the webcam first gets time to adjust to it.
 */
async function takeShot(): Promise<void> {
  ringVisible.value = false;
  lighting.value = true;

  try {
    // The camera starts only once the white screen is actually shown, so it meters with the light on
    await nextPaint();

    if (source.value === 'webcam') {
      await new Promise((resolve) => setTimeout(resolve, WEBCAM_LIGHT_SETTLE_MS));
      const blob = await grabFrame();
      lighting.value = false;
      await captureRepository.create(booth.run!, blob);
    } else {
      await captureRepository.create(booth.run!);
    }
  } finally {
    lighting.value = false;
  }
  booth.recordCapture();

  // The live view only reconnects if another shot follows.
  // After the last one, the sequence moves on to the review.
  if (booth.capturedCount < (booth.imagesPerRun ?? 0)) {
    previewReloadKey.value++;
  }
}

/** Shows the screen version of the photo just taken for SHOT_DISPLAY_MS. */
async function showShot(imageIndex: number): Promise<void> {
  shownShot.value = imageIndex;
  await new Promise((resolve) => setTimeout(resolve, SHOT_DISPLAY_MS));
  shownShot.value = null;
}

/** Resolves once the browser painted the current state of the page. */
async function nextPaint(): Promise<void> {
  return new Promise((resolve) =>
    requestAnimationFrame(() => requestAnimationFrame(() => resolve())),
  );
}

/** Shows the countdown ring and resolves at zero. */
async function runCountdown(): Promise<void> {
  ringVisible.value = true;
  return new Promise((resolve) => countdown.start(COUNTDOWN_SECONDS, () => resolve()));
}

/**
 * Shows a failure with a visible countdown, so the booth does not look frozen,
 * then discards the run and returns to the welcome screen.
 */
async function failAndReturnToWelcome(err: unknown): Promise<void> {
  errorMessage.value = messageFor(err);
  phase.value = 'error';
  await new Promise<void>((resolve) => countdown.start(CAPTURE_ERROR_COUNTDOWN_SECONDS, resolve));
  await booth.abort();
  await router.push({
    name: 'welcome',
  });
}

/** Takes every photo of the run and moves on to the review. */
async function runSequence(): Promise<void> {
  try {
    await runCountdown();
    await takeShot();

    while (booth.capturedCount < (booth.imagesPerRun ?? 0)) {
      await showShot(booth.capturedCount);
      await waitForPreviewReady();
      await runCountdown();
      await takeShot();
    }

    await router.push({
      name: 'review',
    });
  } catch (err) {
    await failAndReturnToWelcome(err);
  }
}

/** Starts the photo sequence. */
function onStartPressed(): void {
  // The sequence runs without the guest, so the inactivity timeout no longer applies
  inactivity.clear();
  phase.value = 'counting';
  void runSequence();
}
</script>

<template>
  <div class="bg-media-backdrop relative flex-1 overflow-hidden">
    <!-- previewContainer is measured for mediaBoxStyle, so it stays an unscaled full-screen box. -->
    <div ref="previewContainer" class="absolute inset-0 flex items-center justify-center">
      <!-- The overlay has to cover the same rectangle as the media instead of fitting itself
           by its own ratio, so both sit in a box sized by mediaBoxStyle. -->
      <div class="relative" :style="mediaBoxStyle">
        <img
          v-if="source !== 'webcam'"
          ref="previewImgEl"
          :src="`${cameraRepository.previewUrl()}?reload=${previewReloadKey}`"
          alt=""
          class="absolute inset-0 h-full w-full object-contain"
          @load="onPreviewLoad"
          @error="onPreviewError"
        />
        <div
          v-if="source !== 'webcam' && !previewLoaded && !lighting"
          class="bg-media-backdrop absolute inset-0 flex items-center justify-center"
        >
          <UIcon name="i-lucide-loader-circle" class="text-primary size-10 animate-spin" />
        </div>
        <video
          v-if="source === 'webcam'"
          ref="videoEl"
          autoplay
          muted
          playsinline
          class="absolute inset-0 h-full w-full object-contain"
          @loadedmetadata="onVideoMetadata"
        ></video>

        <!-- The backend stretches the overlay to the bounds of the capture, so the preview stretches
             it too instead of keeping its ratio. -->
        <img
          v-if="hasOverlay"
          :src="cameraRepository.overlayUrl()"
          alt=""
          class="pointer-events-none absolute inset-0 h-full w-full object-fill"
        />
      </div>
    </div>

    <Transition name="shot">
      <img
        v-if="shownShot !== null && booth.run !== null"
        :src="captureRepository.previewUrl(booth.run, shownShot)"
        alt=""
        class="bg-media-backdrop absolute inset-0 h-full w-full object-contain"
      />
    </Transition>

    <!-- The light uses literal white instead of a theme color, since it lights the guests in both modes. -->
    <Transition name="screen-light">
      <div v-if="lighting" class="absolute inset-0 z-10 flex items-center justify-center bg-white">
        <div class="bg-elevated text-highlighted rounded-full px-8 py-4 text-3xl font-semibold">
          {{ t('pages.capture.smile') }}
        </div>
      </div>
    </Transition>

    <div
      v-if="phase === 'counting'"
      class="absolute inset-0 flex flex-col items-center justify-center gap-6"
    >
      <Transition name="countdown-ring">
        <div
          v-if="ringVisible"
          class="flex h-44 w-44 items-center justify-center rounded-full shadow-lg"
          :style="countdownRingStyle"
        >
          <div class="bg-elevated flex h-36 w-36 flex-col items-center justify-center rounded-full">
            <span class="text-highlighted text-6xl leading-none font-bold tabular-nums">
              {{ countdown.remaining }}
            </span>
            <!-- The photo the countdown is for, so the progress only shows while it counts -->
            <span class="text-muted mt-1 text-sm font-medium tabular-nums">
              {{
                t('pages.capture.progress', { current: upcomingIndex, total: booth.imagesPerRun })
              }}
            </span>
          </div>
        </div>
      </Transition>
    </div>

    <div
      v-if="phase === 'idle' && cameraReady"
      class="bg-scrim/20 absolute inset-0 flex items-center justify-center"
    >
      <UButton
        :label="t('pages.capture.startButton')"
        icon="i-lucide-camera"
        size="xl"
        class="h-20 w-72 justify-center text-2xl"
        @click="onStartPressed"
      />
    </div>

    <div
      v-if="phase === 'error'"
      class="bg-scrim/70 absolute inset-0 flex flex-col items-center justify-center gap-4 px-6 text-center"
    >
      <UIcon name="i-lucide-circle-alert" class="text-error size-12" />
      <p class="text-highlighted text-xl font-medium">{{ errorMessage }}</p>
      <p class="text-muted text-sm">
        {{ t('pages.capture.returningToWelcome', { seconds: countdown.remaining.value }) }}
      </p>
    </div>
  </div>
</template>

<style scoped>
/* The light comes on at once and fades out with an eased decay like a flash. */
.screen-light-leave-active {
  transition: opacity 450ms cubic-bezier(0.25, 0.46, 0.45, 0.94);
}

.screen-light-leave-to {
  opacity: 0;
}

/* Fades the photo just taken in and out. */
.shot-enter-active,
.shot-leave-active {
  transition: opacity 250ms ease-out;
}

.shot-enter-from,
.shot-leave-to {
  opacity: 0;
}

/* Fades the countdown ring in on every appearance. */
.countdown-ring-enter-active {
  transition:
    opacity 250ms ease-out,
    transform 250ms ease-out;
}

.countdown-ring-enter-from {
  opacity: 0;
  transform: scale(0.85);
}

/* Fades the ring out once the flash starts. */
.countdown-ring-leave-active {
  transition: opacity 200ms ease-in;
}

.countdown-ring-leave-to {
  opacity: 0;
}
</style>
