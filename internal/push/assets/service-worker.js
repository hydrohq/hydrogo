self.addEventListener('push', event => {
  let payload = {};
  try { payload = event.data ? event.data.json() : {}; } catch (_) {}
  const title = String(payload.title || 'HydroGo').slice(0, 120);
  const options = { body: String(payload.body || '').slice(0, 1024), data: { url: payload.url || '/' }, icon: '/static/favicon.ico', badge: '/static/favicon.ico' };
  event.waitUntil(self.registration.showNotification(title, options));
});
self.addEventListener('notificationclick', event => {
  event.notification.close();
  const target = new URL(event.notification.data?.url || '/', self.location.origin);
  if (target.origin !== self.location.origin) return;
  event.waitUntil(clients.matchAll({ type: 'window', includeUncontrolled: true }).then(windows => {
    const open = windows.find(client => new URL(client.url).origin === self.location.origin);
    return open ? open.navigate(target.href).then(client => client.focus()) : clients.openWindow(target.href);
  }));
});
