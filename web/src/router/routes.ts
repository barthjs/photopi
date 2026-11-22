import type { RouteLocationNormalizedLoaded, RouteRecordRaw } from 'vue-router';
import AdminLayout from '@/layouts/AdminLayout.vue';
import AppLayout from '@/layouts/AppLayout.vue';
import KioskLayout from '@/layouts/KioskLayout.vue';
// The pages of a guest are part of the main bundle instead of loaded on demand.
// A self-update during a run replaces the chunks of the old build, so a lazy page could no longer load,
// and loading it would also delay the first switch between the screens on a slow device.
import CapturePage from '@/pages/CapturePage.vue';
import ReviewPage from '@/pages/ReviewPage.vue';
import SharePage from '@/pages/SharePage.vue';
import WelcomePage from '@/pages/WelcomePage.vue';

declare module 'vue-router' {
  // eslint-disable-next-line @typescript-eslint/consistent-type-definitions -- augmentation merges into the interface of the library
  interface RouteMeta {
    titleKey?: string;
    // Resolves the interpolation values of the title from the route
    titleParams?: (route: RouteLocationNormalizedLoaded) => Record<string, string>;
    // Nobody reads the page interactively, so it loads a new version as soon as nothing on it would be lost
    unattended?: boolean;
    requiresAuth?: boolean;
    requiresGuest?: boolean;
    // From lg up, the page fills the height of the panel instead of scrolling,
    // for a page with sections only while the route names a section that needs it.
    fillViewport?: boolean | ((route: RouteLocationNormalizedLoaded) => boolean);
  }
}

export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: KioskLayout,
    meta: {
      unattended: true,
    },
    children: [
      {
        path: '',
        name: 'welcome',
        meta: {
          titleKey: 'pages.welcome.title',
        },
        component: WelcomePage,
      },
      {
        path: 'capture',
        name: 'capture',
        meta: {
          titleKey: 'pages.capture.title',
        },
        component: CapturePage,
      },
      {
        path: 'review',
        name: 'review',
        meta: {
          titleKey: 'pages.review.title',
        },
        component: ReviewPage,
      },
      {
        path: 'share',
        name: 'share',
        meta: {
          titleKey: 'pages.share.title',
        },
        component: SharePage,
      },
    ],
  },
  {
    path: '/login',
    component: AppLayout,
    children: [
      {
        path: '',
        name: 'login',
        meta: {
          titleKey: 'pages.login.title',
          requiresGuest: true,
        },
        component: async () => import('@/pages/LoginPage.vue'),
      },
    ],
  },
  {
    path: '/s',
    component: AppLayout,
    children: [
      {
        path: ':token',
        name: 'guestShare',
        meta: {
          titleKey: 'pages.guestShare.title',
        },
        component: async () => import('@/pages/GuestSharePage.vue'),
      },
    ],
  },
  {
    path: '/',
    component: AdminLayout,
    children: [
      {
        path: 'dashboard',
        name: 'dashboard',
        meta: {
          titleKey: 'pages.dashboard.title',
          requiresAuth: true,
        },
        component: async () => import('@/pages/DashboardPage.vue'),
      },
      {
        path: 'config',
        name: 'config',
        meta: {
          titleKey: 'pages.config.title',
          requiresAuth: true,
        },
        component: async () => import('@/pages/ConfigPage.vue'),
      },
      {
        path: 'gallery',
        name: 'gallery',
        meta: {
          titleKey: 'pages.gallery.title',
          requiresAuth: true,
        },
        component: async () => import('@/pages/GalleryPage.vue'),
      },
      {
        path: 'layouts',
        name: 'layouts',
        meta: {
          titleKey: 'pages.layouts.title',
          requiresAuth: true,
          fillViewport: true,
        },
        component: async () => import('@/pages/LayoutsPage.vue'),
      },
      {
        path: 'system',
        name: 'system',
        meta: {
          titleKey: 'pages.system.title',
          requiresAuth: true,
        },
        component: async () => import('@/pages/SystemPage.vue'),
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: {
      name: 'welcome',
    },
  },
];
