<script setup lang="ts">
import { onMounted, reactive, ref, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { ApiError } from '@/api/errors';
import { authRepository } from '@/api/repositories/auth';
import type { AuthLoginRequest } from '@/api/types';
import PasswordField from '@/components/form/PasswordField.vue';
import VirtualKeyboard from '@/components/form/VirtualKeyboard.vue';
import { useInactivityTimer } from '@/composables/useInactivityTimer';
import type { FormErrorTarget } from '@/composables/form/useFormValidation';
import { useMutation } from '@/composables/form/useMutation';
import { redirectTarget } from '@/lib/auth/redirect';
import { FORM_VALIDATE_ON } from '@/lib/form/validation';
import { useAuthStore } from '@/stores/auth';

const { t } = useI18n();
const router = useRouter();
const route = useRoute();

const authStore = useAuthStore();

// The kiosk has no keyboard, a guest who opened the login by accident gets back to the welcome screen
const LOGIN_TIMEOUT_MS = 60_000;

const inactivity = useInactivityTimer(() =>
  router.push({
    name: 'welcome',
  }),
);

onMounted(() => {
  inactivity.arm(LOGIN_TIMEOUT_MS);
});

// The keyboard of the system stays hidden behind the kiosk browser in full screen,
// so a touchscreen gets the on-screen keyboard of the application.
// It opens with the focus on a device that is used by touch and can be toggled on any device.
const keyboardOpen = ref(false);
const touchDevice = window.matchMedia('(pointer: coarse)').matches;

/** onPasswordFocus opens the on-screen keyboard on a device that is used by touch. */
function onPasswordFocus(): void {
  if (touchDevice) keyboardOpen.value = true;
}

/** toggleKeyboard opens or closes the on-screen keyboard. */
function toggleKeyboard(): void {
  keyboardOpen.value = !keyboardOpen.value;
}

// The server accepts an empty password while no admin password is set, so the form has no client validation
const state = reactive<AuthLoginRequest>({
  password: '',
});
const form = useTemplateRef<FormErrorTarget>('form');

const {
  mutate: login,
  pending,
  errorMessage,
} = useMutation(
  async () => {
    try {
      await authRepository.login({
        password: state.password,
      });
    } catch (err) {
      // The server rejects a login with a session that another tab already started, which counts as logged in
      if (err instanceof ApiError && err.status === 409) return;
      throw err;
    }
  },
  {
    form,
    values: state,
    toast: false,
    onSuccess: async () => {
      await authStore.refresh();
      await router.push(
        redirectTarget(route.query, {
          name: 'dashboard',
        }),
      );
    },
  },
);
</script>

<template>
  <div class="relative flex flex-1 items-center justify-center">
    <UButton
      :label="t('common.back')"
      icon="i-lucide-arrow-left"
      color="neutral"
      variant="ghost"
      :to="{ name: 'welcome' }"
      class="absolute top-4 left-4"
    />

    <div class="flex w-full max-w-2xl flex-col items-center gap-3 px-4">
      <UCard class="w-full max-w-sm">
        <!-- The title gives way to the keyboard, which needs the height of the small touchscreen -->
        <template v-if="!keyboardOpen" #header>
          <h1 class="text-highlighted text-center text-xl font-semibold">
            {{ t('pages.login.title') }}
          </h1>
        </template>

        <UForm
          :validate-on="FORM_VALIDATE_ON"
          ref="form"
          :state="state"
          class="space-y-4"
          @submit="login()"
        >
          <UFormField :label="t('fields.password')" name="password">
            <!-- The trailing slot of the field holds the password toggle, so the keyboard toggle sits in the hint -->
            <template #hint>
              <UButton
                icon="i-lucide-keyboard"
                :color="keyboardOpen ? 'primary' : 'neutral'"
                variant="link"
                size="sm"
                :aria-label="t('keyboard.toggle')"
                @click="toggleKeyboard"
              />
            </template>

            <PasswordField
              v-model="state.password"
              autocomplete="current-password"
              :autofocus="!touchDevice"
              :inputmode="keyboardOpen ? 'none' : undefined"
              @focus="onPasswordFocus"
            />
          </UFormField>

          <p v-if="errorMessage" class="text-error text-sm">{{ errorMessage }}</p>

          <UButton type="submit" :label="t('pages.login.submit')" :loading="pending" block />
        </UForm>
      </UCard>

      <VirtualKeyboard v-if="keyboardOpen" v-model="state.password" variant="text" class="w-full" />
    </div>
  </div>
</template>
