(() => {
  if (window.__hydroFormsReady) return;
  window.__hydroFormsReady = true;

  const providers = [
    'gmail.com','googlemail.com','yahoo.com','yahoo.co.uk','yahoo.ca','yahoo.com.au','outlook.com','hotmail.com','hotmail.co.uk','live.com','msn.com','icloud.com','me.com','mac.com','aol.com','proton.me','protonmail.com','pm.me','gmx.com','gmx.de','mail.com','yandex.com','yandex.ru','zoho.com','fastmail.com','hey.com','tuta.com','tutanota.com','qq.com','163.com','126.com','sina.com','sohu.com','comcast.net','att.net','verizon.net','orange.fr','free.fr','laposte.net','wanadoo.fr','btinternet.com','sky.com','virginmedia.com','talktalk.net','bluewin.ch','t-online.de','freenet.de','web.de','mailbox.org','seznam.cz','centrum.cz'
  ];

  const distance = (left, right) => {
    const row = Array.from({ length: right.length + 1 }, (_, i) => i);
    for (let i = 1; i <= left.length; i++) {
      let diagonal = row[0]; row[0] = i;
      for (let j = 1; j <= right.length; j++) {
        const above = row[j];
        row[j] = Math.min(row[j] + 1, row[j - 1] + 1, diagonal + (left[i - 1] === right[j - 1] ? 0 : 1));
        diagonal = above;
      }
    }
    return row[right.length];
  };

  const suggest = input => {
    const output = input.closest('.hydro-form__field')?.querySelector('[data-email-suggestion]');
    if (!output) return;
    output.replaceChildren();
    const match = input.value.trim().match(/^([^@\s]+)@([^@\s]+)$/);
    if (!match || match[1].length > 64 || match[2].length < 5 || match[2].length > 253) return;
    const domain = match[2].toLowerCase();
    let best = null; let bestDistance = 3;
    for (const provider of providers) {
      const current = distance(domain, provider);
      if (current < bestDistance) { best = provider; bestDistance = current; }
    }
    if (!best || best === domain) return;
    const button = document.createElement('button');
    button.type = 'button'; button.textContent = `Use ${match[1]}@${best}`;
    button.addEventListener('click', () => {
      input.value = `${match[1]}@${best}`; input.dispatchEvent(new Event('input', { bubbles: true })); output.replaceChildren(); input.focus();
    }, { once: true });
    output.append(document.createTextNode('Did you mean '), button, document.createTextNode('?'));
  };

  document.addEventListener('input', event => {
    if (event.target.matches('[data-email-suggest]')) suggest(event.target);
  });

  const lock = form => {
    if (form.dataset.submitting) return false;
    form.dataset.submitting = 'true';
    const button = form.querySelector('[data-form-submit]');
    if (button) { button.disabled = true; button.setAttribute('aria-busy', 'true'); }
    form.querySelector('[data-submit-spinner]')?.removeAttribute('hidden');
    const status = form.querySelector('[data-submit-status]');
    if (status) status.textContent = 'Sending…';
    return true;
  };

  document.addEventListener('submit', event => {
    const form = event.target.closest('[data-public-form]');
    if (!form) return;
    if (!lock(form)) { event.preventDefault(); return; }
    form.dataset.htmxPending = form.matches('[hx-post], [hx-get]') ? 'true' : '';
  }, true);
  document.addEventListener('htmx:beforeRequest', event => {
    const form = event.detail?.elt?.closest?.('[data-public-form]');
    if (!form) return;
    if (form.dataset.htmxPending === 'true') { form.dataset.htmxPending = ''; return; }
    if (!lock(form)) event.preventDefault();
  });
  document.addEventListener('htmx:afterRequest', event => {
    const form = event.detail?.elt?.closest?.('[data-public-form]');
    if (!form) return;
    window.setTimeout(() => {
      form.dataset.submitting = '';
      const button = form.querySelector('[data-form-submit]');
      if (button) { button.disabled = false; button.removeAttribute('aria-busy'); }
      form.querySelector('[data-submit-spinner]')?.setAttribute('hidden', '');
    }, 3000);
  });
  window.addEventListener('pageshow', () => document.querySelectorAll('[data-public-form]').forEach(form => {
    form.dataset.submitting = '';
    const button = form.querySelector('[data-form-submit]');
    if (button) { button.disabled = false; button.removeAttribute('aria-busy'); }
    form.querySelector('[data-submit-spinner]')?.setAttribute('hidden', '');
  }));
})();
