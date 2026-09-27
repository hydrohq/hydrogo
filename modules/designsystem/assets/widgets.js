(() => {
  if (window.__hydroDesignSystemWidgets) return;
  window.__hydroDesignSystemWidgets = true;
  document.querySelectorAll('link[data-hydro-late-style]').forEach(link => { link.rel = 'stylesheet'; link.removeAttribute('data-hydro-late-style'); });
  const reduceMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
  const each = (selector, fn, root = document) => root.querySelectorAll(selector).forEach(fn);

  document.addEventListener('input', event => {
    const range = event.target.closest('[data-before-after] input[type="range"]');
    if (range) range.parentElement.style.setProperty('--ds-position', `${Math.max(0, Math.min(100, Number(range.value) || 0))}%`);
    const player = event.target.closest('[data-video-player]');
    if (player && event.target.matches('[data-video-seek]')) {
      const video = player.querySelector('video');
      if (video.duration) video.currentTime = video.duration * Number(event.target.value) / 1000;
    }
    if (player && event.target.matches('[data-video-volume]')) player.querySelector('video').volume = Number(event.target.value);
  });

  const animateCounters = root => {
    if (reduceMotion || !('IntersectionObserver' in window)) return;
    const observer = new IntersectionObserver(entries => entries.forEach(entry => {
      if (!entry.isIntersecting) return;
      observer.unobserve(entry.target);
      each('[data-counter-value]', output => {
        const target = Number(output.dataset.counterValue);
        if (!Number.isFinite(target)) return;
        const start = performance.now();
        output.textContent = '0';
        const step = now => {
          const progress = Math.min(1, (now - start) / 1500);
          const eased = 1 - Math.pow(1 - progress, 3);
          output.textContent = new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 }).format(target * eased);
          if (progress < 1) requestAnimationFrame(step);
        };
        requestAnimationFrame(step);
      }, entry.target);
    }), { threshold: .25 });
    each('[data-counter-group]', group => observer.observe(group), root);
  };

  document.addEventListener('click', async event => {
    const copy = event.target.closest('[data-copy-text]');
    if (copy) {
      const value = copy.dataset.copyText || '';
      try {
        if (navigator.clipboard && window.isSecureContext) await navigator.clipboard.writeText(value);
        else {
          const input = document.createElement('textarea'); input.value = value; input.setAttribute('readonly', '');
          input.style.position = 'fixed'; input.style.opacity = '0'; document.body.append(input); input.select();
          const copied = document.execCommand('copy'); input.remove(); if (!copied) throw new Error('copy failed');
        }
        copy.querySelector('[data-copy-idle]')?.setAttribute('hidden', '');
        copy.querySelector('[data-copy-done]')?.removeAttribute('hidden');
        copy.setAttribute('aria-busy', 'true');
        window.setTimeout(() => {
          copy.querySelector('[data-copy-idle]')?.removeAttribute('hidden');
          copy.querySelector('[data-copy-done]')?.setAttribute('hidden', ''); copy.removeAttribute('aria-busy');
        }, 2000);
      } catch (_) { copy.dispatchEvent(new CustomEvent('hydro:copy-error', { bubbles: true })); }
    }
    const videoButton = event.target.closest('[data-video-player] button');
    if (videoButton) {
      const player = videoButton.closest('[data-video-player]'); const video = player.querySelector('video');
      if (videoButton.matches('[data-video-play]')) video.paused ? video.play() : video.pause();
      if (videoButton.matches('[data-video-mute]')) video.muted = !video.muted;
      if (videoButton.matches('[data-video-pip]') && document.pictureInPictureEnabled) video.requestPictureInPicture().catch(() => {});
      if (videoButton.matches('[data-video-fullscreen]')) (document.fullscreenElement ? document.exitFullscreen() : player.requestFullscreen()).catch?.(() => {});
    }
  });

  document.addEventListener('change', event => {
    const speed = event.target.closest('[data-video-speed]');
    if (speed) speed.closest('[data-video-player]').querySelector('video').playbackRate = Number(speed.value);
  });

  const initVideos = root => each('[data-video-player]', player => {
    if (player.dataset.videoReady) return;
    player.dataset.videoReady = 'true'; player.classList.add('is-enhanced');
    const video = player.querySelector('video'); video.controls = false;
    const play = player.querySelector('[data-video-play]'); const mute = player.querySelector('[data-video-mute]');
    const seek = player.querySelector('[data-video-seek]'); const time = player.querySelector('[data-video-time]');
    const clock = value => { if (!Number.isFinite(value)) return '0:00'; const m = Math.floor(value / 60); return `${m}:${String(Math.floor(value % 60)).padStart(2, '0')}`; };
    const update = () => {
      play.textContent = video.paused ? 'Play' : 'Pause'; play.setAttribute('aria-label', video.paused ? 'Play video' : 'Pause video');
      mute.textContent = video.muted || video.volume === 0 ? 'Unmute' : 'Mute';
      if (video.duration) seek.value = String(Math.round(video.currentTime / video.duration * 1000));
      time.textContent = `${clock(video.currentTime)} / ${clock(video.duration)}`;
    };
    ['play', 'pause', 'timeupdate', 'durationchange', 'volumechange'].forEach(name => video.addEventListener(name, update));
    player.addEventListener('keydown', event => {
      if (event.target.matches('input,select,button')) return;
      if (event.code === 'Space') { event.preventDefault(); video.paused ? video.play() : video.pause(); }
      if (event.key.toLowerCase() === 'm') video.muted = !video.muted;
      if (event.key.toLowerCase() === 'f') document.fullscreenElement ? document.exitFullscreen() : player.requestFullscreen();
      if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') { event.preventDefault(); video.currentTime = Math.max(0, video.currentTime + (event.key === 'ArrowLeft' ? -5 : 5)); }
    });
    update();
  }, root);

  const initSticky = root => each('[data-sticky-header]', header => {
    if (header.dataset.stickyReady) return;
    header.dataset.stickyReady = 'true';
    const threshold = Math.max(0, Number(header.dataset.threshold) || 80);
    let previous = window.scrollY; let queued = false;
    const update = () => {
      const current = window.scrollY; header.classList.toggle('is-scrolled', current > 0);
      if (current < threshold || current < previous || header.contains(document.activeElement)) {
        header.classList.remove('is-hidden'); header.inert = false;
      } else if (current > previous) { header.classList.add('is-hidden'); header.inert = true; }
      previous = current; queued = false;
    };
    window.addEventListener('scroll', () => { if (!queued) { queued = true; requestAnimationFrame(update); } }, { passive: true });
  }, root);

  const celebrate = target => {
    if (reduceMotion || !target) return;
    const host = target.matches('[data-confetti-success]') ? target : target.querySelector('[data-confetti-success]');
    if (!host) return;
    for (let i = 0; i < 20; i++) {
      const piece = document.createElement('i'); piece.className = 'ds-confetti__piece';
      piece.style.setProperty('--ds-hue', String((i * 47) % 360)); piece.style.setProperty('--ds-x', `${Math.round(Math.random() * 360 - 180)}px`);
      piece.style.setProperty('--ds-y', `${Math.round(80 + Math.random() * 180)}px`); host.append(piece);
      window.setTimeout(() => piece.remove(), 1200);
    }
  };
  document.addEventListener('hydro:form-success', event => celebrate(event.target));
  document.addEventListener('htmx:afterSwap', event => {
    const status = event.detail?.xhr?.status;
    if (status >= 200 && status < 300) celebrate(event.detail.target);
  });
  document.addEventListener('click', event => {
    const action = event.target.closest('[data-contact-action]');
    if (action) {
      action.dispatchEvent(new CustomEvent('hydro:contact-clicked', { bubbles: true, detail: { action: action.dataset.contactAction } }));
      if (navigator.sendBeacon) navigator.sendBeacon('/analytics/contact', new URLSearchParams({ action: action.dataset.contactAction }));
    }
  });

  const init = root => { animateCounters(root); initVideos(root); initSticky(root); };
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', () => init(document)); else init(document);
  document.addEventListener('htmx:afterSettle', event => init(event.target));
})();
