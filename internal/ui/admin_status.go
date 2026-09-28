package ui

const AdminStatusFilterCSS = `.status-pill{display:inline-flex;align-items:center;gap:.4rem;padding:.2rem .55rem;border:1px solid var(--line);border-radius:999px;background:var(--surface);color:var(--muted);font:600 .75rem/1.3 system-ui,sans-serif;cursor:pointer}.status-pill:hover,.status-pill[aria-pressed=true]{box-shadow:0 0 0 2px color-mix(in srgb,currentColor 18%,transparent)}.status-pill--published{color:var(--green,#15803d);background:var(--green-subtle,#f0fdf4);border-color:color-mix(in srgb,var(--green,#15803d) 30%,transparent)}.status-pill--draft{color:var(--amber,#a16207);background:var(--amber-subtle,#fffbeb);border-color:color-mix(in srgb,var(--amber,#a16207) 30%,transparent)}.status-pill--scheduled{color:var(--blue,#1d4ed8);background:var(--blue-subtle,#eff6ff);border-color:color-mix(in srgb,var(--blue,#1d4ed8) 30%,transparent)}.status-pill--archived{color:var(--muted);background:var(--bg-subtle);border-color:var(--line)}.status-filter-status{display:flex;align-items:center;gap:.75rem;margin:0 0 .6rem;color:var(--muted);font-size:.8rem}.status-filter-status button{border:0;background:none;color:var(--link);text-decoration:underline;cursor:pointer}`

const AdminStatusFilterJS = `(()=>{
  const table = () => document.querySelector('table[data-admin-list-state]');
  const apply = (active, syncURL) => {
    const currentTable = table(); if (!currentTable) return;
    const parent = currentTable.parentElement;
    let panel = parent.querySelector('[data-status-filter-panel]');
    if (!panel) {
      panel = document.createElement('div'); panel.className = 'status-filter-status'; panel.dataset.statusFilterPanel = '';
      const message = document.createElement('span'); message.setAttribute('role', 'status'); message.setAttribute('aria-live', 'polite'); message.dataset.statusFilterMessage = '';
      const clear = document.createElement('button'); clear.type = 'button'; clear.dataset.statusFilterClear = ''; clear.textContent = 'Clear filter'; clear.hidden = true;
      panel.append(message, clear); parent.insertBefore(panel, currentTable);
    }
    currentTable.dataset.activeStatus = active;
    currentTable.querySelectorAll('tbody tr[data-status-row]').forEach(row => { row.hidden = !!active && row.dataset.statusRow !== active; });
    currentTable.querySelectorAll('[data-status-filter]').forEach(button => button.setAttribute('aria-pressed', String(!!active && button.dataset.statusFilter === active)));
    panel.querySelector('[data-status-filter-message]').textContent = active ? 'Filtered to ' + active : 'Showing all statuses';
    panel.querySelector('[data-status-filter-clear]').hidden = !active;
    if (syncURL) { const url = new URL(location.href); if (active) url.searchParams.set('status', active); else url.searchParams.delete('status'); history.replaceState(history.state, '', url); }
  };
  const initialize = () => apply(new URL(location.href).searchParams.get('status') || '', false);
  document.addEventListener('click', event => {
    const pill = event.target.closest('[data-status-filter]'); const clear = event.target.closest('[data-status-filter-clear]');
    if (!pill && !clear) return;
    if (pill && !table()?.contains(pill)) return;
    const current = table()?.dataset.activeStatus || '';
    apply(clear ? '' : (current === pill.dataset.statusFilter ? '' : pill.dataset.statusFilter), true);
  });
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', initialize, {once:true}); else initialize();
  document.addEventListener('hydro:admin-navigated', initialize);
})();`

const AdminListStateJS = `(()=>{
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
})();`

const AdminCopyTextJS = `(()=>{document.addEventListener('click',async event=>{const button=event.target.closest('[data-copy-text]');if(!button)return;const status=button.parentElement.querySelector('[data-copy-status]');try{if(navigator.clipboard&&window.isSecureContext){await navigator.clipboard.writeText(button.dataset.copyText)}else{const field=document.createElement('textarea');field.value=button.dataset.copyText;field.setAttribute('readonly','');field.style.position='fixed';field.style.opacity='0';document.body.append(field);field.select();const copied=document.execCommand('copy');field.remove();if(!copied)throw new Error('copy unavailable')}if(status)status.textContent='Copied'}catch{if(status)status.textContent='Could not copy'}})})();`
