(() => {
  const button = document.querySelector('[data-push-subscribe]');
  if (!button) return;
  const unsubscribe = document.querySelector('[data-push-unsubscribe]');
  const status = document.querySelector('[data-push-status]');
  const csrf = document.querySelector('meta[name="csrf-token"]')?.content || '';
  const decode = value => Uint8Array.from(atob(value.replace(/-/g, '+').replace(/_/g, '/')), c => c.charCodeAt(0));
  button.addEventListener('click', async () => {
    button.disabled = true;
    try {
      if (!('serviceWorker' in navigator) || !('PushManager' in window)) throw new Error('This browser does not support push notifications.');
      const permission = await Notification.requestPermission();
      if (permission !== 'granted') throw new Error('Notification permission was not granted.');
      const config = await fetch('/api/v1/push/config', { credentials: 'same-origin' }).then(r => r.ok ? r.json() : Promise.reject());
      if (!config.public_key) throw new Error('Browser notifications are not configured.');
      const registration = await navigator.serviceWorker.register('/push/service-worker.js', { scope: '/' });
      const subscription = await registration.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: decode(config.public_key) });
      const response = await fetch('/api/v1/push/web/subscription', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify(subscription) });
      if (!response.ok) throw new Error('Subscription could not be saved.');
      status.textContent = 'Browser notifications are enabled.';
    } catch (error) { status.textContent = error.message || 'Push setup failed.'; }
    finally { button.disabled = false; }
  });
  unsubscribe?.addEventListener('click', async () => {
    unsubscribe.disabled = true;
    try {
      const registration = await navigator.serviceWorker.getRegistration('/');
      const subscription = await registration?.pushManager.getSubscription();
      if (!subscription) { status.textContent = 'No browser notification subscription is active.'; return; }
      const response = await fetch('/api/v1/push/web/subscription', { method: 'DELETE', credentials: 'same-origin', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf }, body: JSON.stringify({ endpoint: subscription.endpoint }) });
      if (!response.ok) throw new Error('Subscription could not be removed.');
      await subscription.unsubscribe();
      status.textContent = 'Browser notifications are disabled.';
    } catch (error) { status.textContent = error.message || 'Push setup failed.'; }
    finally { unsubscribe.disabled = false; }
  });
})();
