package ui

const AdminStatusFilterCSS = `.status-pill{display:inline-flex;align-items:center;gap:.4rem;padding:.2rem .55rem;border:1px solid var(--line);border-radius:999px;background:var(--surface);color:var(--muted);font:500 .72rem/1.3 system-ui,sans-serif;user-select:none;pointer-events:none}.status-pill--published{color:color-mix(in srgb,var(--green,#15803d) 75%,var(--ink));background:var(--green-subtle,#f0fdf4);border-color:color-mix(in srgb,var(--green,#15803d) 20%,transparent)}.status-pill--draft{color:color-mix(in srgb,var(--amber,#a16207) 75%,var(--ink));background:var(--amber-subtle,#fffbeb);border-color:color-mix(in srgb,var(--amber,#a16207) 20%,transparent)}.status-pill--scheduled{color:color-mix(in srgb,var(--blue,#1d4ed8) 75%,var(--ink));background:var(--blue-subtle,#eff6ff);border-color:color-mix(in srgb,var(--blue,#1d4ed8) 20%,transparent)}.status-pill--archived{color:var(--muted);background:var(--bg-subtle);border-color:var(--line)}.status-filter-status[hidden],[data-status-filter-panel][hidden]{display:none!important}.status-filter-status{display:inline-flex;align-items:center;gap:.75rem;margin:0 0 .75rem;padding:.4rem .85rem;background:var(--bg-subtle);border:1px solid var(--line);border-radius:var(--radius-md);color:var(--ink-secondary);font-size:.78rem}.status-filter-status [data-status-filter-message]{font-weight:600}.status-filter-status button{display:inline-flex;align-items:center;gap:.3rem;padding:.2rem .55rem;border:1px solid var(--line);border-radius:var(--radius-sm);background:var(--surface);color:var(--muted);font-size:.74rem;font-weight:600;cursor:pointer;transition:var(--transition);text-decoration:none}.status-filter-status button:hover{color:var(--ink);border-color:var(--line-light);background:var(--surface-hover)}`

const AdminStatusFilterJS = `(()=>{
  const table = () => document.querySelector('table[data-admin-list-state]');
  const labels = {
    published: 'Yayında',
    draft: 'Taslak',
    scheduled: 'Zamanlanmış',
    archived: 'Arşivlenmiş',
    retracted: 'Geri Çekilmiş'
  };
  const apply = (active, syncURL) => {
    const currentTable = table(); if (!currentTable) return;
    const host = currentTable.closest('.table-wrap') || currentTable;
    const parent = host.parentElement || currentTable.parentElement;
    let panel = parent.querySelector('[data-status-filter-panel]');
    if (!active) {
      if (panel) {
        panel.hidden = true;
        panel.style.display = 'none';
      }
      currentTable.dataset.activeStatus = '';
      currentTable.querySelectorAll('tbody tr[data-status-row]').forEach(row => { row.hidden = false; });
      currentTable.querySelectorAll('[data-status-filter]').forEach(button => button.setAttribute('aria-pressed', 'false'));
      if (syncURL) { const url = new URL(location.href); url.searchParams.delete('status'); history.replaceState(history.state, '', url); }
      return;
    }
    if (!panel) {
      panel = document.createElement('div'); panel.className = 'status-filter-status'; panel.dataset.statusFilterPanel = '';
      const message = document.createElement('span'); message.setAttribute('role', 'status'); message.setAttribute('aria-live', 'polite'); message.dataset.statusFilterMessage = '';
      const clear = document.createElement('button'); clear.type = 'button'; clear.dataset.statusFilterClear = ''; clear.textContent = '✕ Filtreyi Temizle'; clear.hidden = false;
      panel.append(message, clear);
      parent.insertBefore(panel, host);
    }
    currentTable.dataset.activeStatus = active;
    currentTable.querySelectorAll('tbody tr[data-status-row]').forEach(row => { row.hidden = row.dataset.statusRow !== active; });
    currentTable.querySelectorAll('[data-status-filter]').forEach(button => button.setAttribute('aria-pressed', String(button.dataset.statusFilter === active)));
    const displayLabel = labels[active] || active;
    panel.querySelector('[data-status-filter-message]').textContent = 'Duruma göre filtrelendi: ' + displayLabel;
    panel.querySelector('[data-status-filter-clear]').hidden = false;
    panel.hidden = false;
    panel.style.display = 'inline-flex';
    if (syncURL) { const url = new URL(location.href); url.searchParams.set('status', active); history.replaceState(history.state, '', url); }
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
