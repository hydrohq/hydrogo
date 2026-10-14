// Newsroom news pages: headline slider, category filter, live search, copy
// link and cover fallbacks. Everything is wired through data attributes so the
// pages need no inline handlers under the strict CSP.
(function () {
  'use strict';

  var SLIDE_ACTIVE = 'w-9 h-8 flex items-center justify-center font-black text-xs rounded bg-[#ed1c24] text-white scale-110 shadow cursor-pointer';
  var SLIDE_IDLE = 'w-9 h-8 flex items-center justify-center font-bold text-xs rounded bg-slate-900/80 text-white hover:bg-slate-700 transition cursor-pointer';
  var PILL_ACTIVE = 'filter-pill px-3.5 py-1.5 rounded-full text-xs font-bold bg-[#ed1c24] text-white shadow-sm transition';
  var PILL_IDLE = 'filter-pill px-3.5 py-1.5 rounded-full text-xs font-bold bg-white text-slate-700 hover:bg-slate-200 border border-slate-300 transition';

  // Cover images fall back to the original upload when a CDN variant fails.
  document.addEventListener('error', function (event) {
    var image = event.target;
    if (!image || image.tagName !== 'IMG' || !image.dataset.fallbackSrc) return;
    var fallback = image.dataset.fallbackSrc;
    delete image.dataset.fallbackSrc;
    if (image.getAttribute('src') !== fallback) image.src = fallback;
  }, true);

  function norm(value) {
    if (!value) return '';
    return value.toLocaleLowerCase('tr').replace(/ğ/g, 'g').replace(/ü/g, 'u').replace(/ş/g, 's')
      .replace(/ı/g, 'i').replace(/ö/g, 'o').replace(/ç/g, 'c').trim();
  }

  function initSlider() {
    var buttons = Array.prototype.slice.call(document.querySelectorAll('[data-slide-btn]'));
    var box = document.querySelector('[data-news-slider]');
    if (!box || buttons.length === 0) return;
    var current = 0;
    var timer = null;

    function show(index) {
      var button = buttons[index];
      if (!button) return;
      current = index;
      var image = document.getElementById('slider-main-img');
      var tag = document.getElementById('slider-main-tag');
      var title = document.getElementById('slider-main-title');
      var link = document.getElementById('slider-main-link');
      var excerpt = document.getElementById('slider-main-excerpt');
      if (image && button.dataset.img) image.src = button.dataset.img;
      if (tag && button.dataset.cat) tag.textContent = button.dataset.cat;
      if (title && button.dataset.title) {
        title.textContent = button.dataset.title;
        title.href = button.dataset.url;
      }
      if (link && button.dataset.url) link.href = button.dataset.url;
      if (excerpt) excerpt.textContent = button.dataset.excerpt || '';
      buttons.forEach(function (other) { other.className = SLIDE_IDLE; });
      button.className = SLIDE_ACTIVE;
    }

    function start() {
      if (timer) clearInterval(timer);
      if (buttons.length > 1) {
        timer = setInterval(function () { show((current + 1) % buttons.length); }, 4500);
      }
    }

    buttons.forEach(function (button, index) {
      button.addEventListener('click', function () { show(index); start(); });
    });
    box.addEventListener('mouseenter', function () { clearInterval(timer); });
    box.addEventListener('mouseleave', start);
    if (!window.matchMedia || !window.matchMedia('(prefers-reduced-motion: reduce)').matches) start();
  }

  function cards() {
    return Array.prototype.slice.call(document.querySelectorAll('[data-news-card]'));
  }

  function showCount(visible) {
    var counter = document.getElementById('news-counter');
    if (counter) counter.textContent = '(' + visible + ')';
    var empty = document.getElementById('no-news-alert');
    if (empty) empty.hidden = visible !== 0;
  }

  function selectCategory(category) {
    var key = norm(category) || 'all';
    document.querySelectorAll('.filter-pill[data-filter-pill]').forEach(function (pill) {
      pill.className = norm(pill.dataset.filterPill) === key ? PILL_ACTIVE : PILL_IDLE;
    });
    document.querySelectorAll('#main-nav-links [data-news-category]').forEach(function (link) {
      var active = norm(link.dataset.newsCategory) === key;
      link.classList.toggle('bg-red-800', active);
      link.classList.toggle('text-white', active);
      link.classList.toggle('hover:bg-red-700', !active);
    });
    var visible = 0;
    cards().forEach(function (card) {
      var show = key === 'all' || norm(card.dataset.cat) === key;
      card.style.display = show ? 'flex' : 'none';
      if (show) visible++;
    });
    showCount(visible);
    try {
      var url = new URL(window.location.href);
      if (key === 'all') url.searchParams.delete('category');
      else url.searchParams.set('category', key);
      window.history.replaceState({}, '', url);
    } catch (e) {}
  }

  function initFilters() {
    if (!document.getElementById('cards-container')) return;
    document.addEventListener('click', function (event) {
      var target = event.target.closest('[data-filter-pill], [data-news-category]');
      if (!target) return;
      event.preventDefault();
      selectCategory(target.dataset.filterPill || target.dataset.newsCategory);
      if (target.dataset.newsCategory) {
        var grid = document.getElementById('news-grid');
        if (grid) grid.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }
    });
    var search = document.querySelector('[data-news-search]');
    if (search) {
      search.addEventListener('input', function () {
        var query = norm(search.value);
        var visible = 0;
        cards().forEach(function (card) {
          var show = !query || norm(card.dataset.title).indexOf(query) !== -1 || norm(card.dataset.cat).indexOf(query) !== -1;
          card.style.display = show ? 'flex' : 'none';
          if (show) visible++;
        });
        showCount(visible);
      });
    }
    var category = new URLSearchParams(window.location.search).get('category');
    if (category) selectCategory(category);
  }

  function initCopyLink() {
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

  function init() {
    initSlider();
    initFilters();
    initCopyLink();
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init);
  else init();
})();
