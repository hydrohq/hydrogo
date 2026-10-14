// Corporate news pages: cover fallbacks and the copy-link button, wired through
// data attributes so the pages need no inline handlers under the strict CSP.
(function () {
  'use strict';

  document.addEventListener('error', function (event) {
    var image = event.target;
    if (!image || image.tagName !== 'IMG' || !image.dataset.fallbackSrc) return;
    var fallback = image.dataset.fallbackSrc;
    delete image.dataset.fallbackSrc;
    if (image.getAttribute('src') !== fallback) image.src = fallback;
  }, true);

  function init() {
    document.querySelectorAll('[data-copy-link]').forEach(function (button) {
      button.addEventListener('click', function () {
        if (!navigator.clipboard) return;
        navigator.clipboard.writeText(window.location.href).then(function () {
          var label = button.querySelector('[data-copy-link-label]');
          if (!label || !button.dataset.copiedLabel) return;
          var original = label.textContent;
          label.textContent = button.dataset.copiedLabel;
          setTimeout(function () { label.textContent = original; }, 2000);
        }).catch(function () {});
      });
    });
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init);
  else init();
})();
