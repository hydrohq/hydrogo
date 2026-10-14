// Member sign-in, registration and account behaviour. Served from
// /member/assets/member-auth.js so theme pages need no inline scripts.
// Message text comes from data-msg-* attributes on the form so each theme can
// word it; the English strings below are only fallbacks.
(function () {
  'use strict';

  function message(form, key, fallback) {
    var value = form && form.getAttribute('data-msg-' + key);
    return value || fallback;
  }

  var COMMON = ['password', 'qwerty', 'welcome', 'admin', 'secret', 'letmein', 'iloveyou', 'monkey', 'dragon', 'football', 'abc123'];

  // Mirrors the server's entropy rule so weak passwords are caught before submit.
  function passwordScore(value) {
    var chars = Array.from(value);
    var lower = value.toLowerCase();
    var pool = 0;
    var classes = 0;
    if (chars.length < 8 || chars.length > 1024 || new Set(chars).size < 4) return 0;
    if (COMMON.some(function (word) { return lower.indexOf(word) !== -1; })) return 0;
    if (/[a-z]/.test(value)) { pool += 26; classes++; }
    if (/[A-Z]/.test(value)) { pool += 26; classes++; }
    if (/[0-9]/.test(value)) { pool += 10; classes++; }
    if (/[^A-Za-z0-9]/.test(value)) { pool += 32; classes++; }
    var entropy = chars.length * Math.log2(pool || 1);
    return classes && entropy >= 40 ? Math.min(4, 1 + Math.floor((entropy - 40) / 20)) : 0;
  }

  function initPasswordStrength(form) {
    var input = form.querySelector('[data-password-strength]');
    var meter = form.querySelector('[data-password-meter]');
    var feedback = form.querySelector('[data-password-feedback]');
    if (!input || !meter) return;
    input.addEventListener('input', function () {
      var strength = passwordScore(input.value);
      meter.value = strength;
      if (!feedback) return;
      feedback.textContent = strength
        ? message(form, 'strength', 'Password strength: ') + (strength >= 3 ? message(form, 'strong', 'Strong') : message(form, 'ok', 'Acceptable')) + '.'
        : message(form, 'hint', 'Use at least 8 characters; avoid common passwords and mix character types.');
    });
    form.addEventListener('submit', function (event) {
      if (passwordScore(input.value) !== 0) return;
      event.preventDefault();
      event.stopImmediatePropagation();
      if (feedback) feedback.textContent = message(form, 'weak', 'Choose a longer, harder to guess password.');
      input.focus();
    }, true);
  }

  function initAuthForm(form) {
    var status = form.querySelector('[data-member-status]');
    function show(type, text) {
      if (!status) return;
      status.className = 'member-auth-status status-' + type;
      status.textContent = text;
      status.style.display = 'block';
    }
    form.addEventListener('submit', function (event) {
      event.preventDefault();
      var button = form.querySelector('button[type="submit"]');
      var label = button ? button.innerHTML : '';
      if (button) {
        button.disabled = true;
        button.textContent = message(form, 'busy', 'Working…');
      }
      if (status) {
        status.style.display = 'none';
        status.textContent = '';
      }
      function restore() {
        if (!button) return;
        button.disabled = false;
        button.innerHTML = label;
      }
      var csrf = form.querySelector('input[name="_csrf"]');
      fetch(form.action, {
        method: 'POST',
        credentials: 'same-origin',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded',
          'X-CSRF-Token': csrf ? csrf.value : '',
          'Accept': 'application/json'
        },
        body: new URLSearchParams(new FormData(form))
      }).then(function (response) {
        return response.text().then(function (raw) {
          var data = null;
          try { data = JSON.parse(raw); } catch (e) {}
          if (!response.ok) {
            var detail = ((data && data.error) || raw || '').toLowerCase();
            if (detail.indexOf('already registered') !== -1 || response.status === 409) show('error', message(form, 'exists', 'This email is already registered. Try signing in.'));
            else if (detail.indexOf('invalid credentials') !== -1 || response.status === 401) show('error', message(form, 'credentials', 'Email or password is incorrect.'));
            else if (detail.indexOf('too many') !== -1 || response.status === 429) show('error', message(form, 'rate', 'Too many attempts. Please wait and try again.'));
            else if (detail.indexOf('csrf') !== -1 || response.status === 403) show('error', message(form, 'csrf', 'Security check failed. Please reload the page.'));
            else show('error', message(form, 'failed', 'The request could not be completed.'));
            restore();
            return;
          }
          var target = data && data.redirect ? data.redirect : form.querySelector('input[name="return_to"]').value;
          show('success', message(form, 'success', 'Success! Redirecting…'));
          setTimeout(function () { window.location.href = target; }, 200);
        });
      }).catch(function () {
        show('error', message(form, 'network', 'Connection error: the server could not be reached.'));
        restore();
      });
    });
  }

  function initLogout() {
    document.querySelectorAll('[data-member-logout]').forEach(function (button) {
      button.addEventListener('click', function () {
        fetch('/member/logout', {
          method: 'POST',
          credentials: 'same-origin',
          headers: { 'X-CSRF-Token': button.getAttribute('data-csrf') || '' }
        }).then(function () {
          window.location.assign('/member/login');
        });
      });
    });
  }

  function init() {
    document.querySelectorAll('[data-member-form]').forEach(function (form) {
      initPasswordStrength(form);
      initAuthForm(form);
    });
    initLogout();
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init);
  else init();
})();
