(()=>{
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
})();
