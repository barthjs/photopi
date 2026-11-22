import { existsSync, readFileSync } from 'node:fs';
import { fileURLToPath, URL } from 'node:url';

import { defineConfig } from 'vite';
import tailwindcss from '@tailwindcss/vite';
import vue from '@vitejs/plugin-vue';
import ui from '@nuxt/ui/vite';

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
  plugins: [tailwindcss(), vue(), ui(uiOptions)],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    host: true,
    https: devHttps(),
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
      '/readyz': {
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
