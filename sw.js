const CACHE_NAME = '{{GIT_REV}}';
const urlsToCache = [
    '/index.html',
    '/manifest.json',
    '/jquery.min.js',
    'https://cdn.jsdelivr.net/gh/beercss/beercss@v3.11.33/dist/cdn/beer.min.css',
    'https://cdn.jsdelivr.net/npm/material-icons@1.13.14/iconfont/material-icons.min.css'
];

self.addEventListener('install', (event) => {
    event.waitUntil(
        caches.open(CACHE_NAME)
            .then((cache) => {
                return cache.addAll(urlsToCache);
            })
    );
});

self.addEventListener('fetch', (event) => {
    event.respondWith(
        caches.match(event.request)
            .then((response) => {
                if (response) {
                    return response;
                }
                return fetch(event.request);
            })
    );
});

self.addEventListener('activate', (event) => {
    event.waitUntil(
        caches.keys().then((cacheNames) => {
            return Promise.all(
                cacheNames.map((cacheName) => {
                    if (cacheName !== CACHE_NAME) {
                        return caches.delete(cacheName);
                    }
                })
            );
        })
    );
});
