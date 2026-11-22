import { existsSync, readFileSync } from 'node:fs';
import { fileURLToPath, URL } from 'node:url';

import { defineConfig } from 'vite';
import tailwindcss from '@tailwindcss/vite';
import vue from '@vitejs/plugin-vue';
import ui from '@nuxt/ui/vite';

import { pwa } from './pwa/plugin.ts';
import uiOptions from './ui.config.ts';

/** The optional certificate of the dev server, which serves HTTPS once both files exist. */
const DEV_CERTS = {
  key: fileURLToPath(new URL('./.certs/key.pem', import.meta.url)),
  cert: fileURLToPath(new URL('./.certs/cert.pem', import.meta.url)),
};

/** Returns the certificate of the dev server, or undefined to serve plain HTTP. */
function devHttps(): { key: Buffer; cert: Buffer } | undefined {
  if (!existsSync(DEV_CERTS.key) || !existsSync(DEV_CERTS.cert)) {
    return undefined;
  }

  return {
    key: readFileSync(DEV_CERTS.key),
    cert: readFileSync(DEV_CERTS.cert),
  };
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    tailwindcss(),
    vue(),
    ui(uiOptions),
    pwa({
      srcDir: fileURLToPath(new URL('./src', import.meta.url)),
      locales: {
        de: fileURLToPath(new URL('./src/locales/de.json', import.meta.url)),
        en: fileURLToPath(new URL('./src/locales/en.json', import.meta.url)),
      },
      defaultLanguage: 'en',
      // The default colors of the theme, since the service worker knows no configured ones
      theme: {
        light: {
          background: '#ffffff',
          text: '#1e293b',
          muted: '#475569',
          primary: '#0f766e',
          onPrimary: '#ffffff',
        },
        dark: {
          background: '#0f172a',
          text: '#f1f5f9',
          muted: '#cbd5e1',
          primary: '#14b8a6',
          onPrimary: '#020617',
        },
      },
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    host: true,
    https: devHttps(),
    proxy: {
      // ws forwards the upgrade of the realtime connection as well
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
        ws: true,
      },
      // The backend writes the manifest with the name and the colors of the configuration
      '/site.webmanifest': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      // The backend serves the documentation from the last build in docs/dist
      '/docs': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
});
