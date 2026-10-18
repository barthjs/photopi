<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';
import QRCode from 'qrcode';

import { ApiError } from '@/api/errors';
import { shareRepository } from '@/api/repositories/share';
import type { ShareEmailRequest } from '@/api/types';
import VirtualKeyboard from '@/components/form/VirtualKeyboard.vue';
import { useApiError } from '@/composables/useApiError';
import { useFormValidation, type FormErrorTarget } from '@/composables/form/useFormValidation';
import { useInactivityTimer } from '@/composables/useInactivityTimer';
import { useMutation } from '@/composables/form/useMutation';
import { useShareJob } from '@/composables/useShareJob';
import { FORM_VALIDATE_ON } from '@/lib/form/validation';
import { useBoothStore } from '@/stores/booth';
import { useMetaStore } from '@/stores/meta';

const GENERAL_TIMEOUT_MS = 90_000;
const DISABLED_TIMEOUT_MS = 30_000;
const SUCCESS_TIMEOUT_MS = 10_000;
const LOCKOUT_TIMEOUT_MS = 60_000;

const { t } = useI18n();
const router = useRouter();
const booth = useBoothStore();
const metaStore = useMetaStore();
const { messageFor } = useApiError();

const emailEnabled = computed(() => metaStore.meta?.share.emailEnabled ?? false);
const cloudEnabled = computed(() => metaStore.meta?.share.cloudEnabled ?? false);
const printEnabled = computed(() => metaStore.meta?.share.printEnabled ?? false);
const sharingConfigured = computed(
  () => emailEnabled.value || cloudEnabled.value || printEnabled.value,
);

const inactivity = useInactivityTimer(() =>
  router.push({
    name: 'welcome',
  }),
);

const emailState = reactive<ShareEmailRequest>({
  email: '',
});
const emailForm = useTemplateRef<FormErrorTarget>('emailForm');

// The server also checks the format of the address, the client only that it is given.
const { validate: validateEmail } = useFormValidation<ShareEmailRequest>(() => ({
  email: {
    required: 'required',
  },
}));
const emailError = ref('');
const emailFocused = ref(false);
const emailJob = useShareJob(() => shareRepository.getEmail(booth.run as number));
const emailSent = computed(() => emailJob.phase.value === 'done');

/** focusEmail shows the virtual keyboard for the address field. */
function focusEmail(): void {
  emailFocused.value = true;
}

/** unfocusEmail hides the virtual keyboard. */
function unfocusEmail(): void {
  emailFocused.value = false;
}

/** emailFailedMessage returns the message for a failed email, naming the admin contact if there is one. */
function emailFailedMessage(): string {
  const adminEmail = metaStore.meta?.share.adminEmail;
  return adminEmail
    ? t('pages.share.emailFailedContact', {
        adminEmail,
        run: booth.run,
      })
    : t('pages.share.emailFailed');
}

const { mutate: sendEmail, pending: sendingEmail } = useMutation(
  () =>
    shareRepository.sendEmail(booth.run as number, {
      email: emailState.email,
    }),
  {
    form: emailForm,
    values: emailState,
    toast: false,
    onSuccess: () => {
      emailError.value = '';
      emailFocused.value = false;
      emailJob.track();
    },
    onError: (err) => {
      // An email of this run is already on its way, its state is all that is left to follow
      if (err instanceof ApiError && err.code === 'conflict') {
        emailJob.track();
        return true;
      }

      emailError.value =
        err instanceof ApiError && err.code === 'rate_limited'
          ? messageFor(err)
          : emailFailedMessage();

      // The backend limits the attempts per run. Another attempt would fail the same way,
      // so the page returns to the welcome screen sooner.
      if (err instanceof ApiError && err.code === 'rate_limited') {
        inactivity.arm(LOCKOUT_TIMEOUT_MS);
      }
      return true;
    },
  },
);

const sending = computed(() => sendingEmail.value || emailJob.phase.value === 'pending');

watch(emailJob.phase, (phase) => {
  if (phase === 'done') {
    inactivity.arm(SUCCESS_TIMEOUT_MS);
  }
  if (phase === 'failed') {
    emailError.value =
      emailJob.reason.value === 'unreachable'
        ? t('pages.share.emailUnreachable')
        : emailFailedMessage();
  }
});

const printJob = useShareJob(() => shareRepository.getPrint(booth.run as number));
const printSent = computed(() => printJob.phase.value === 'done');

const { mutate: submitPrint, pending: submittingPrint } = useMutation(
  () => shareRepository.print(booth.run as number),
  {
    toast: false,
    onSuccess: () => {
      printJob.track();
    },
    onError: (err) => {
      // The printer is not ready, the guest may try again once it is
      if (err instanceof ApiError && err.code === 'conflict') {
        printJob.fail('queue_unavailable');
        return true;
      }

      // The backend limits the attempts per run, so a retry would fail the same way
      if (err instanceof ApiError && err.code === 'rate_limited') {
        inactivity.arm(LOCKOUT_TIMEOUT_MS);
      }
      printJob.fail('failed');
      return true;
    },
  },
);

const printing = computed(() => submittingPrint.value || printJob.phase.value === 'pending');

const printMessage = computed(() => {
  switch (printJob.phase.value) {
    case 'pending':
      return t('pages.share.printPending');
    case 'done':
      return t('pages.share.printSuccess');
    case 'failed':
      return printJob.reason.value === 'queue_unavailable'
        ? t('pages.share.printQueueUnavailable')
        : t('pages.share.printFailed');
    default:
      return '';
  }
});

const cloudJob = useShareJob(() => shareRepository.getCloudUpload(booth.run as number));
const qrCanvas = ref<HTMLCanvasElement | null>(null);

watch(cloudJob.url, async (url) => {
  if (!url) return;
  await nextTick();
  if (qrCanvas.value)
    await QRCode.toCanvas(qrCanvas.value, url, {
      width: 176,
      margin: 1,
    });
});

/** startCloudUpload starts sharing the run as a link and follows it until the QR code can be shown. */
async function startCloudUpload(run: number): Promise<void> {
  try {
    await shareRepository.startCloudUpload(run);
    cloudJob.track();
  } catch {
    cloudJob.fail('failed');
  }
}

onMounted(async () => {
  if (booth.run === null) {
    await router.replace({
      name: 'welcome',
    });
    return;
  }

  if (!sharingConfigured.value) {
    inactivity.arm(DISABLED_TIMEOUT_MS);
    return;
  }

  inactivity.arm(GENERAL_TIMEOUT_MS);

  if (cloudEnabled.value) {
    await startCloudUpload(booth.run);
  }
});

const showFinishDialog = ref(false);

/** openFinishDialog asks before the guest finishes. */
function openFinishDialog(): void {
  showFinishDialog.value = true;
}

/** onFinish returns to the welcome screen. */
function onFinish(): void {
  showFinishDialog.value = false;
  void router.push({
    name: 'welcome',
  });
}
</script>

<template>
  <div class="relative flex h-full flex-col items-center overflow-hidden p-4">
    <div
      v-if="!sharingConfigured"
      class="flex flex-1 flex-col items-center justify-center gap-3 text-center"
    >
      <UIcon name="i-lucide-heart" class="text-primary size-12" />
      <p class="text-highlighted text-2xl font-semibold">{{ t('pages.share.disabledMessage') }}</p>
    </div>

    <template v-else>
      <div class="flex w-full max-w-2xl flex-1 flex-col items-center gap-3 overflow-hidden">
        <p v-if="emailEnabled" class="text-highlighted shrink-0 text-center text-lg font-medium">
          {{
            emailSent
              ? t('pages.share.emailSuccess', { email: emailState.email })
              : t('pages.share.emailPrompt')
          }}
        </p>

        <UForm
          :validate-on="FORM_VALIDATE_ON"
          ref="emailForm"
          :state="emailState"
          :validate="validateEmail"
          class="flex w-full shrink-0 flex-col items-center gap-3"
          @submit="sendEmail()"
        >
          <UFormField
            v-if="emailEnabled && !emailSent"
            name="email"
            :ui="{ error: 'text-center text-base font-medium' }"
            class="w-full shrink-0"
          >
            <UInput
              :model-value="emailState.email"
              readonly
              icon="i-lucide-mail"
              size="xl"
              :placeholder="t('pages.share.emailPrompt')"
              class="w-full cursor-pointer [&_input]:cursor-pointer"
              @focus="focusEmail"
              @click="focusEmail"
            />
          </UFormField>
          <p v-if="emailError" class="text-error shrink-0 text-center text-base font-medium">
            {{ emailError }}
          </p>

          <div class="flex w-full shrink-0 items-center justify-center gap-4">
            <UButton
              :label="t('pages.share.finish')"
              color="neutral"
              variant="outline"
              @click="openFinishDialog"
            />
            <UButton
              v-if="emailEnabled && !emailSent"
              :label="t('pages.share.emailSend')"
              icon="i-lucide-send"
              type="submit"
              :loading="sending"
            />
            <UButton
              v-if="printEnabled && !printSent"
              :label="t('pages.share.printSend')"
              icon="i-lucide-printer"
              :loading="printing"
              @click="submitPrint()"
            />
          </div>
          <p
            v-if="printEnabled && printMessage"
            class="text-muted shrink-0 text-center text-base font-medium"
            :class="{ 'text-error': printJob.phase.value === 'failed' }"
          >
            {{ printMessage }}
          </p>
        </UForm>

        <div class="relative min-h-0 w-full flex-1">
          <div
            v-if="cloudEnabled"
            class="absolute inset-0 flex flex-col items-center justify-center gap-2"
          >
            <template v-if="cloudJob.phase.value === 'pending'">
              <UIcon name="i-lucide-loader-circle" class="text-primary size-10 animate-spin" />
              <p class="text-muted text-sm">{{ t('pages.share.cloudPreparing') }}</p>
            </template>
            <template v-else-if="cloudJob.phase.value === 'done'">
              <canvas ref="qrCanvas" class="rounded-lg" />
              <p class="text-muted text-sm">{{ t('pages.share.cloudQrLabel') }}</p>
            </template>
            <template v-else-if="cloudJob.phase.value === 'failed'">
              <UIcon name="i-lucide-circle-alert" class="text-error size-10" />
              <p class="text-muted text-sm">
                {{
                  cloudJob.reason.value === 'unreachable'
                    ? t('pages.share.cloudUnreachable')
                    : t('pages.share.cloudFailed')
                }}
              </p>
            </template>
          </div>

          <div v-if="emailFocused" class="absolute inset-0 z-10 flex items-center justify-center">
            <div
              class="bg-elevated ring-default flex w-full flex-col items-center gap-1 rounded-2xl p-2 ring-1"
            >
              <div class="flex w-full shrink-0 justify-center">
                <UButton
                  icon="i-lucide-chevron-down"
                  size="xs"
                  color="neutral"
                  variant="ghost"
                  :aria-label="t('keyboard.hide')"
                  @click="unfocusEmail"
                />
              </div>
              <VirtualKeyboard v-model="emailState.email" class="w-full" />
            </div>
          </div>
        </div>
      </div>
    </template>

    <UModal
      v-model:open="showFinishDialog"
      :title="t('pages.share.finishDialog.title')"
      :description="t('pages.share.finishDialog.text')"
      :ui="{ footer: 'justify-end' }"
    >
      <template #footer="{ close }">
        <UButton
          :label="t('pages.share.finishDialog.back')"
          color="neutral"
          variant="outline"
          @click="close"
        />
        <UButton :label="t('pages.share.finishDialog.confirm')" @click="onFinish" />
      </template>
    </UModal>
  </div>
</template>
