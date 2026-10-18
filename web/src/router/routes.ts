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
  interface RouteMeta {
    titleKey?: string;
    // Resolves the interpolation values of the title from the route.
    titleParams?: (route: RouteLocationNormalizedLoaded) => Record<string, string>;
    requiresAuth?: boolean;
    requiresGuest?: boolean;
    // The page uses the whole width of the main area instead of the container width.
    fullWidth?: boolean;
    // From lg up, the page fills the height between header and footer instead of scrolling.
    fillViewport?: boolean;
  }
}

export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: KioskLayout,
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
        component: () => import('@/pages/LoginPage.vue'),
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
        component: () => import('@/pages/GuestSharePage.vue'),
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
        component: () => import('@/pages/DashboardPage.vue'),
      },
      {
        path: 'config',
        name: 'config',
        meta: {
          titleKey: 'pages.config.title',
          requiresAuth: true,
        },
        component: () => import('@/pages/ConfigPage.vue'),
      },
      {
        path: 'gallery',
        name: 'gallery',
        meta: {
          titleKey: 'pages.gallery.title',
          requiresAuth: true,
        },
        component: () => import('@/pages/GalleryPage.vue'),
      },
      {
        path: 'layouts',
        name: 'layouts',
        meta: {
          titleKey: 'pages.layouts.title',
          requiresAuth: true,
          fullWidth: true,
          fillViewport: true,
        },
        component: () => import('@/pages/LayoutsPage.vue'),
      },
      {
        path: 'system',
        name: 'system',
        meta: {
          titleKey: 'pages.system.title',
          requiresAuth: true,
        },
        component: () => import('@/pages/SystemPage.vue'),
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
