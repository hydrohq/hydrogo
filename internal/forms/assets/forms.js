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
    if (event.target.matches('[data-character-counter]')) {
      const counter = event.target.closest('.hydro-form__field')?.querySelector('[data-character-count]');
      if (counter) counter.textContent = `${event.target.value.length} / ${event.target.maxLength}`;
    }
    if (event.target.matches('[data-phone-mask]')) {
      const input = event.target;
      const digitsBefore = input.value.slice(0, input.selectionStart || 0).replace(/\D/g, '').length;
      const digits = input.value.replace(/\D/g, '').slice(0, 15);
      const international = input.value.trimStart().startsWith('+');
      const groups = digits.match(/.{1,3}/g) || [];
      input.value = `${international ? '+' : ''}${groups.join(' ')}`;
      let cursor = international ? 1 : 0, seen = 0;
      while (cursor < input.value.length && seen < digitsBefore) { if (/\d/.test(input.value[cursor])) seen++; cursor++; }
      input.setSelectionRange(cursor, cursor);
    }
  });

  document.addEventListener('click', event => {
    const close = event.target.closest('[data-policy-close]');
    if (close) {
      close.closest('dialog')?.close();
      return;
    }
    const trigger = event.target.closest('[data-policy-open]');
    if (!trigger) return;
    const dialog = document.getElementById(trigger.dataset.policyOpen);
    if (dialog instanceof HTMLDialogElement && !dialog.open) dialog.showModal();
  });

  document.querySelectorAll('[data-cascade-select]').forEach(select => {
    const form = select.closest('form');
    const parent = form?.elements.namedItem(select.dataset.cascadeParent);
    if (!parent) return;
    let choices = {};
    try { choices = JSON.parse(select.dataset.cascadeOptions || '{}'); } catch {}
    const update = () => {
      const previous = select.value;
      select.replaceChildren(new Option('Choose one', ''));
      (choices[parent.value] || []).forEach(value => select.add(new Option(value, value)));
      select.value = (choices[parent.value] || []).includes(previous) ? previous : '';
      select.disabled = !(choices[parent.value] || []).length;
    };
    parent.addEventListener('change', update);
    update();
  });

  document.querySelectorAll('form[data-step-wizard]').forEach(form => {
    const fields = Array.from(form.querySelectorAll('[data-form-step]'));
    const count = Math.max(1, Number(form.dataset.stepCount) || 1);
    const previous = form.querySelector('[data-form-previous]');
    const next = form.querySelector('[data-form-next]');
    const status = form.querySelector('[data-form-step-status]');
    let current = 1;
    const show = step => {
      current = Math.max(1, Math.min(count, step));
      fields.forEach(field => { field.hidden = Number(field.dataset.formStep) !== current; });
      if (previous) previous.disabled = current === 1;
      if (next) next.hidden = current === count;
      if (status) status.textContent = `Step ${current} of ${count}`;
      fields.find(field => Number(field.dataset.formStep) === current)?.querySelector('input,select,textarea')?.focus({ preventScroll: true });
    };
    previous?.addEventListener('click', () => show(current - 1));
    form.addEventListener('hydro:reset-step', () => show(1));
    next?.addEventListener('click', () => {
      const active = fields.filter(field => Number(field.dataset.formStep) === current);
      const invalid = active.flatMap(field => Array.from(field.querySelectorAll('input,select,textarea'))).find(control => !control.checkValidity());
      if (invalid) { invalid.reportValidity(); return; }
      show(current + 1);
    });
    form.addEventListener('submit', event => {
      const invalid = fields.flatMap(field => Array.from(field.querySelectorAll('input,select,textarea'))).find(control => !control.checkValidity());
      if (!invalid) return;
      event.preventDefault();
      show(Number(invalid.closest('[data-form-step]')?.dataset.formStep) || 1);
      invalid.reportValidity();
    }, true);
    show(1);
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
    if (!form.matches('[hx-post], [hx-get]') && form.hasAttribute('data-reset-on-success') && window.fetch) {
      event.preventDefault();
      const status = form.querySelector('[data-submit-status]');
      fetch(form.action, { method: form.method || 'POST', body: new FormData(form), credentials: 'same-origin', headers: { Accept: 'application/json' } })
        .then(response => {
          if (!response.ok) throw new Error('submission rejected');
          form.reset();
          form.querySelectorAll('[data-character-counter]').forEach(input => {
            const counter = input.closest('.hydro-form__field')?.querySelector('[data-character-count]');
            if (counter) counter.textContent = `0 / ${input.maxLength}`;
          });
          form.dispatchEvent(new Event('hydro:reset-step'));
          form.dataset.success = 'true';
          if (status) status.textContent = 'Sent successfully.';
        })
        .catch(() => { if (status) status.textContent = 'We could not send your message. Please try again.'; })
        .finally(() => {
          form.dataset.submitting = '';
          const button = form.querySelector('[data-form-submit]');
          if (button) { button.disabled = false; button.removeAttribute('aria-busy'); }
          form.querySelector('[data-submit-spinner]')?.setAttribute('hidden', '');
        });
    }
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
    const statusCode = Number(event.detail?.xhr?.status || 0);
    const succeeded = event.detail?.successful === true || (statusCode >= 200 && statusCode < 300);
    if (succeeded && form.hasAttribute('data-reset-on-success')) {
      form.reset();
      form.querySelectorAll('[data-character-counter]').forEach(input => {
        const counter = input.closest('.hydro-form__field')?.querySelector('[data-character-count]');
        if (counter) counter.textContent = `0 / ${input.maxLength}`;
      });
      form.dispatchEvent(new Event('hydro:reset-step'));
      form.dataset.success = 'true';
    }
    if (succeeded && form.hasAttribute('data-conversion-enabled') && form.dataset.conversionSent !== 'true') {
      form.dataset.conversionSent = 'true';
      const conversionID = form.dataset.googleAdsConversionId;
      if (conversionID && typeof window.gtag === 'function') {
        const label = form.dataset.googleAdsConversionLabel;
        window.gtag('event', 'conversion', { send_to: label ? `${conversionID}/${label}` : conversionID });
      }
      const eventName = form.dataset.metaPixelEvent;
      if (eventName && typeof window.fbq === 'function') window.fbq('track', eventName);
    }
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

  const exitPrompts = Array.from(document.querySelectorAll('[data-exit-intent]'));
  if (exitPrompts.length && matchMedia('(hover: hover) and (pointer: fine)').matches && innerWidth > 768) {
    const shown = new Set();
    const open = prompt => {
      const key = prompt.dataset.exitIntentKey;
      const storageKey = `hydro:exit-intent:${key}`;
      const cooldown = Math.max(0, Number(prompt.dataset.exitIntentCooldown) || 604800000);
      let lastShown = 0;
      try { lastShown = Number(localStorage.getItem(storageKey)) || 0; } catch {}
      if (shown.has(key) || Date.now() - lastShown < cooldown) return;
      shown.add(key);
      try { localStorage.setItem(storageKey, String(Date.now())); } catch {}
      prompt.hidden = false;
      prompt._hydroPreviousFocus = document.activeElement;
      prompt.querySelector('input:not([type="hidden"]),textarea,select,button')?.focus();
    };
    const close = prompt => {
      prompt.hidden = true;
      prompt._hydroPreviousFocus?.focus?.();
    };
    document.addEventListener('mouseout', event => {
      if (event.clientY > 0 || event.relatedTarget !== null) return;
      const prompt = exitPrompts.find(candidate => candidate.hidden);
      if (prompt) open(prompt);
    });
    document.addEventListener('click', event => {
      const prompt = event.target.closest('[data-exit-intent]');
      if (!prompt || prompt.hidden) return;
      if (event.target === prompt || event.target.closest('[data-exit-intent-close]')) close(prompt);
    });
    document.addEventListener('keydown', event => {
      if (event.key !== 'Escape') return;
      const prompt = exitPrompts.find(candidate => !candidate.hidden);
      if (prompt) close(prompt);
    });
  }
})();
