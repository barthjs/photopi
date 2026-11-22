<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';
import type { NavigationMenuItem } from '@nuxt/ui';

import { useMutation } from '@/composables/form/useMutation';
import DashboardFrame from '@/components/dashboard/DashboardFrame.vue';
import AppVersion from '@/components/system/AppVersion.vue';
import FooterLinks, { type FooterLink } from '@/components/system/FooterLinks.vue';
import { docsUrl } from '@/lib/app/docs';
import { useAuthStore } from '@/stores/auth';
import { useMetaStore } from '@/stores/meta';

const REPOSITORY_URL = 'https://github.com/barthjs/photopi';

const { t } = useI18n();

// The icon links of the footer of the sidebar
const footerLinks = computed<FooterLink[]>(() => [
  {
    label: t('common.docs'),
    icon: 'i-lucide-book-open',
    to: docsUrl(),
    external: true,
  },
  {
    label: t('common.repository'),
    icon: 'i-lucide-github',
    to: REPOSITORY_URL,
    external: true,
  },
]);
const route = useRoute();
const router = useRouter();

const authStore = useAuthStore();
const metaStore = useMetaStore();

// From lg up, a page that fills the viewport keeps its height inside the panel, so its table scrolls on its own
const fillViewport = computed(() => {
  const fill = route.meta.fillViewport;
  return typeof fill === 'function' ? fill(route) : !!fill;
});

const appName = computed(() => metaStore.meta?.name || 'PhotoPi');

const navItems = computed<NavigationMenuItem[]>(() => [
  {
    label: t('admin.nav.dashboard'),
    icon: 'i-lucide-layout-dashboard',
    to: {
      name: 'dashboard',
    },
  },
  {
    label: t('admin.nav.gallery'),
    icon: 'i-lucide-images',
    to: {
      name: 'gallery',
    },
  },
  {
    label: t('admin.nav.layouts'),
    icon: 'i-lucide-layout-grid',
    to: {
      name: 'layouts',
    },
  },
  {
    label: t('admin.nav.system'),
    icon: 'i-lucide-server',
    to: {
      name: 'system',
    },
  },
]);

// The configuration sits at the bottom of the sidebar, apart from the pages of the photo booth
const bottomNavItems = computed<NavigationMenuItem[]>(() => [
  {
    label: t('admin.nav.config'),
    icon: 'i-lucide-settings',
    to: {
      name: 'config',
    },
  },
]);

const { mutate: logout, pending: loggingOut } = useMutation(async () => authStore.logout(), {
  onSuccess: () => {
    void router.push({
      name: 'welcome',
    });
  },
});
</script>

<template>
  <DashboardFrame
    :items="navItems"
    :bottom-items="bottomNavItems"
    :home-to="{
      name: 'welcome',
    }"
    storage-key="admin"
    :desktop-navbar="false"
  >
    <template #logo>
      <span class="text-highlighted font-semibold">{{ appName }}</span>
    </template>

    <template #footer="{ collapsed }">
      <div class="flex w-full flex-col gap-2">
        <div class="flex items-center gap-1" :class="collapsed ? 'flex-col' : ''">
          <UTooltip :text="t('admin.logout')" :disabled="!collapsed">
            <UButton
              :label="collapsed ? undefined : t('admin.logout')"
              :aria-label="t('admin.logout')"
              icon="i-lucide-log-out"
              color="neutral"
              variant="ghost"
              :loading="loggingOut"
              @click="logout()"
            />
          </UTooltip>
          <UTooltip :text="t('admin.openApp')">
            <UButton
              icon="i-lucide-external-link"
              color="neutral"
              variant="ghost"
              :aria-label="t('admin.openApp')"
              :to="{ name: 'welcome' }"
              :class="collapsed ? '' : 'ms-auto'"
            />
          </UTooltip>
          <UColorModeSwitch />
        </div>
        <div class="flex w-full items-center" :class="collapsed ? 'flex-col' : 'justify-between'">
          <AppVersion v-if="!collapsed" :repository-url="REPOSITORY_URL" />
          <FooterLinks :links="footerLinks" :class="collapsed ? 'flex-col' : ''" />
        </div>
      </div>
    </template>

    <div class="flex flex-col" :class="{ 'lg:min-h-0 lg:flex-1': fillViewport }">
      <RouterView />
    </div>
  </DashboardFrame>
</template>
