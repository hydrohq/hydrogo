(()=>{
  const list = () => document.querySelector('table[data-admin-list-state]');
  const key = path => 'hydro-admin-list-state:' + path;
  const save = table => { try { sessionStorage.setItem(key(location.pathname), location.search); } catch {} };
  const restore = () => {
    const table = list(); if (!table) return;
    let saved = ''; try { saved = sessionStorage.getItem(key(location.pathname)) || ''; } catch {}
    if (location.search) { save(table); return; }
    if (!saved) return;
    try { if (new URL(document.referrer).origin !== location.origin) return; } catch { return; }
    location.replace(location.pathname + saved + location.hash);
  };
  document.addEventListener('click', event => {
    if (!list()) return;
    const link = event.target.closest('a[href]'); if (!link) return;
    const next = new URL(link.href, location.href);
    if (next.origin === location.origin && next.pathname !== location.pathname) save(list());
  });
  window.addEventListener('pagehide', () => { if (list()) save(list()); });
  const ready = () => restore();
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', ready, {once:true}); else ready();
  document.addEventListener('hydro:admin-navigated', restore);
})();
