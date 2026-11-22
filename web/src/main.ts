import '@/assets/css/main.css';

import { createApp } from 'vue';
import { createPinia } from 'pinia';
import ui from '@nuxt/ui/vue-plugin';

import App from '@/App.vue';
import { setupAppRecovery } from '@/appRecovery';
import { setAuthenticationRequiredHandler } from '@/api/client/client';
import { loginLocation } from '@/lib/auth/redirect';
import { applyThemeColors } from '@/lib/app/theme';
import { i18n, setLocaleFromBackend } from '@/i18n';
import { router } from '@/router';
import { useAuthStore } from '@/stores/auth';
import { useCameraHealthStore } from '@/stores/cameraHealth';
import { useMetaStore } from '@/stores/meta';

const app = createApp(App);
const pinia = createPinia();

app.use(pinia);
app.use(i18n);
app.use(router);
app.use(ui);

setupAppRecovery(app);

// Only an expired or missing admin session bounces to the login page
setAuthenticationRequiredHandler(() => {
  useAuthStore().clearSession();
  void router.push(
    loginLocation(
      {
        name: 'login',
      },
      router.currentRoute.value.fullPath,
    ),
  );
});

useCameraHealthStore().start();

app.mount('#app');

void useMetaStore()
  .load()
  .then((meta) => {
    setLocaleFromBackend(meta.language);
    applyThemeColors(meta.theme);
  })
  .catch((err: unknown) => {
    console.error(err);
  });
