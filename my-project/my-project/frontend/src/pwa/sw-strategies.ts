// Workbox runtime cache matrix (research R7). Imported by vite.config.ts —
// the app shell itself is precached (Cache-First, build-versioned).
// SWR for profile/progress reads; Cache-First for static media;
// Network-Only (default, no rule) for auth and speech uploads.
import type { RuntimeCaching } from 'workbox-build';

export const runtimeCaching: RuntimeCaching[] = [
  {
    // Profile, tracks, dashboard, review queue: stale-while-revalidate.
    urlPattern: /\/api\/v1\/(me|tracks|dashboard|review-queue)$/,
    handler: 'StaleWhileRevalidate',
    method: 'GET',
    options: {
      cacheName: 'api-swr',
      expiration: { maxEntries: 32, maxAgeSeconds: 7 * 24 * 60 * 60 },
      cacheableResponse: { statuses: [200] },
    },
  },
  {
    // Lesson content: SWR (text payloads of the current track).
    urlPattern: /\/api\/v1\/lessons\/.+$/,
    handler: 'StaleWhileRevalidate',
    method: 'GET',
    options: {
      cacheName: 'api-lessons',
      expiration: { maxEntries: 64, maxAgeSeconds: 7 * 24 * 60 * 60 },
      cacheableResponse: { statuses: [200] },
    },
  },
  {
    // Audio, images, fonts: Cache-First.
    urlPattern: /\.(?:mp3|ogg|wav|png|jpg|jpeg|svg|webp|woff2?)$/,
    handler: 'CacheFirst',
    method: 'GET',
    options: {
      cacheName: 'static-media',
      expiration: { maxEntries: 128, maxAgeSeconds: 30 * 24 * 60 * 60 },
      cacheableResponse: { statuses: [200] },
    },
  },
];
