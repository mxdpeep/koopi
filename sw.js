'use strict';
const CACHE_NAME = '{{GIT_REV}}';
const URLS = [
    '/manifest.json',
    '/jquery.min.js',
    'https://cdn.jsdelivr.net/gh/beercss/beercss@v3.13.3/dist/cdn/beer.min.css',
    'https://cdn.jsdelivr.net/npm/material-icons@1.13.14/iconfont/material-icons.min.css'
];

self.addEventListener('install', (event) => {
    self.skipWaiting();
    event.waitUntil(
        caches.open(CACHE_NAME)
            .then((cache) => {
                return cache.addAll(URLS);
            })
    );
});

self.addEventListener('message', (event) => {
    if (event.data && event.data.type === 'SKIP_WAITING') {
        self.skipWaiting();
    }
});

self.addEventListener('fetch', (event) => {
    if (event.request.method !== 'GET') return;
    const url = new URL(event.request.url);

    if (url.pathname.endsWith('meta.json')) {
        event.respondWith(fetch(event.request)); 
        return;
    }
    if (url.pathname.endsWith('index.html')) {
        event.respondWith(fetch(event.request)); 
        return;
    }

    event.respondWith(
        caches.match(event.request).then((response) => {
            return response || fetch(event.request).catch(() => {
                return new Response('<!DOCTYPE html><html><head><meta charset="UTF-8"><meta http-equiv="X-UA-Compatible" content="IE=edge"><meta name="viewport" content="width=device-width,initial-scale=1.0"></head><body style="text-align:center"><h1 style="font-size:8rem">🛸</h1><h2 style="text-align:center">chybí internetové připojení</h2><h3 style="text-align:center"><a rel="noopener nofollow" style="color:red;text-decoration:none;font-weight:bold" href="javascript:location.reload();"><br>klikni pro refreš ↻</a></h3></body></html>', { status: 503 });
            });
        })
    );
});

self.addEventListener('activate', (event) => {
    event.waitUntil(
        Promise.all([
            self.clients.claim(),
            caches.keys().then((cacheNames) => {
                return Promise.all(
                    cacheNames.map((cacheName) => {
                        if (cacheName !== CACHE_NAME) {
                            return caches.delete(cacheName);
                        }
                    })
                );
            })
        ])
    );
});