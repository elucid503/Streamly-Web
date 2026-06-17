const CACHE = 'streamly-v1';

self.addEventListener('install', (e) => {

  e.waitUntil(self.skipWaiting());

});

self.addEventListener('activate', (e) => {

  e.waitUntil(
    caches.keys()
      .then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k))))
      .then(() => self.clients.claim())
  );

});

self.addEventListener('fetch', (e) => {

  if (e.request.method !== 'GET') return;

  const url = new URL(e.request.url);

  // Never cache API responses
  if (url.pathname.startsWith('/api/')) return;

  if (e.request.mode === 'navigate') {

    e.respondWith(
      fetch(e.request).catch(() => caches.match('/'))
    );

    return;

  }

  e.respondWith(
    caches.match(e.request).then((cached) => {

      if (cached) return cached;

      return fetch(e.request).then((response) => {

        if (response.ok) {

          const clone = response.clone();
          caches.open(CACHE).then((cache) => cache.put(e.request, clone));

        }

        return response;

      });

    })
  );

});
