// Applies the saved colour scheme before first paint. External (not inline) so
// the admin CSP can stay at script-src 'self'.
(function () {
  try {
    var t = localStorage.getItem('hydrogo-theme') || 'system';
    var d = t === 'dark' || (t === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches);
    if (d) document.documentElement.classList.add('dark');
  } catch (e) {}
})();
