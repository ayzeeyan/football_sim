/*
 * Service worker (F14): installable shell plus a read-only offline cache.
 *
 * - APP_SHELL: the built application shell, cached on install so the app
 *   opens with the server down.
 * - DATA: a last-good snapshot of same-origin GET /api responses. The
 *   network is always authoritative; the cache only answers when the
 *   network fails, and it is never written from the client. Simulation
 *   (POST) and mutation requests are never cached or replayed.
 */
const VERSION = 'v1';
const APP_SHELL = `topfive-shell-${VERSION}`;
const DATA = `topfive-data-${VERSION}`;

const SHELL_ASSETS = ['/', '/index.html', '/manifest.webmanifest', '/icon.svg'];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(APP_SHELL).then((cache) => cache.addAll(SHELL_ASSETS)).then(() => self.skipWaiting()),
  );
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) =>
        Promise.all(keys.filter((key) => key !== APP_SHELL && key !== DATA).map((key) => caches.delete(key))),
      )
      .then(() => self.clients.claim()),
  );
});

self.addEventListener('fetch', (event) => {
  const request = event.request;
  if (request.method !== 'GET') {
    // Mutations go straight to the network; offline they fail loudly.
    return;
  }
  const url = new URL(request.url);
  if (url.origin !== self.location.origin) {
    return;
  }

  // Read-only API data: network-first, last-good cache fallback.
  if (url.pathname.startsWith('/api/')) {
    event.respondWith(
      fetch(request)
        .then((response) => {
          if (response.ok) {
            const clone = response.clone();
            caches.open(DATA).then((cache) => cache.put(request, clone));
          }
          return response;
        })
        .catch(() =>
          caches.open(DATA).then((cache) => cache.match(request)).then((cached) => {
            if (cached) {
              const headers = new Headers(cached.headers);
              headers.set('X-Offline-Cache', 'hit');
              return new Response(cached.body, { status: cached.status, headers });
            }
            return new Response(JSON.stringify({ error: 'offline' }), {
              status: 503,
              headers: { 'Content-Type': 'application/json' },
            });
          }),
        ),
    );
    return;
  }

  // App shell and hashed assets: cache-first, refresh in the background.
  event.respondWith(
    caches.match(request).then((cached) => {
      const network = fetch(request)
        .then((response) => {
          if (response.ok) {
            const clone = response.clone();
            caches.open(APP_SHELL).then((cache) => cache.put(request, clone));
          }
          return response;
        })
        .catch(() => cached);
      return cached || network;
    }),
  );
});
