(function () {
  var channel = null;
  try {
    if (typeof BroadcastChannel !== 'undefined') {
      channel = new BroadcastChannel('hydro_member_auth');
    }
  } catch (e) {}

  function setCurrentMember(member) {
    return member && member.email ? member : null;
  }

  function renderHeader(member) {
    var loginLinks = document.querySelectorAll('a[href="/member/login"], a[href^="/member/login?"], .nav-member-badge');
    if (!loginLinks.length) return;

    if (member && member.email) {
      loginLinks.forEach(function (link) {
        if (link.classList.contains('nav-member-badge')) {
          link.textContent = '';
          var icon = document.createElement('span');
          icon.className = 'nav-member-icon';
          icon.textContent = '👤';
          link.appendChild(icon);
          link.appendChild(document.createTextNode(' ' + member.email));
          return;
        }
        var badge = document.createElement('a');
        badge.href = '/member/account';
        badge.className = 'nav-member-badge is-active';
        var icon = document.createElement('span');
        icon.className = 'nav-member-icon';
        icon.textContent = '👤';
        badge.appendChild(icon);
        badge.appendChild(document.createTextNode(' ' + member.email));
        if (link.parentNode) {
          link.parentNode.replaceChild(badge, link);
        }
      });
    } else {
      loginLinks.forEach(function (link) {
        if (link.classList.contains('nav-member-badge')) {
          var a = document.createElement('a');
          a.href = '/member/login';
          a.className = 'nav-item nav-login-link';
          a.textContent = 'Giriş';
          if (link.parentNode) {
            link.parentNode.replaceChild(a, link);
          }
        }
      });
    }
  }

  function verifySessionWithServer() {
    fetch('/member/me', {
      headers: { 'Accept': 'application/json' },
      credentials: 'same-origin'
    })
      .then(function (res) {
        if (res.ok) return res.json();
        if (res.status === 401 || res.status === 403) {
          var unauthenticated = new Error('unauthenticated');
          unauthenticated.code = 'UNAUTHENTICATED';
          throw unauthenticated;
        }
        throw new Error('unavailable');
      })
      .then(function (data) {
        if (data && data.email) {
          var member = { email: data.email, id: data.id };
          setCurrentMember(member);
          renderHeader(member);
        }
      })
      .catch(function (err) {
        if (err && err.code === 'UNAUTHENTICATED') {
          setCurrentMember(null);
          renderHeader(null);
        }
      });
  }

  function highlightActiveNav() {
    var path = window.location.pathname;
    var navLinks = document.querySelectorAll('.news-nav a');
    navLinks.forEach(function (link) {
      var href = link.getAttribute('href');
      if (!href) return;
      if (href === path || (href !== '/' && path.startsWith(href)) || (href === '/' && (path === '/' || path === '/public/pages/tr/demo/html'))) {
        link.classList.add('is-active');
      }
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', highlightActiveNav);
  } else {
    highlightActiveNav();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', verifySessionWithServer);
  } else {
    verifySessionWithServer();
  }

  if (channel) {
    channel.onmessage = function (event) {
      if (event && event.data) {
        if (event.data.type === 'LOGIN') {
          setCurrentMember(event.data.member);
          renderHeader(event.data.member);
        } else if (event.data.type === 'LOGOUT') {
          setCurrentMember(null);
          renderHeader(null);
        }
      }
    };
  }

  window.hydroAuth = {
    setMember: function (member) {
      setCurrentMember(member);
      renderHeader(member);
      if (channel) channel.postMessage({ type: 'LOGIN', member: member });
    },
    clearMember: function () {
      setCurrentMember(null);
      renderHeader(null);
      if (channel) channel.postMessage({ type: 'LOGOUT' });
    }
  };
})();

// ==========================================================================
// UX / Scroll / Animation Controllers
// ==========================================================================
(function () {
  function initUX() {
    // Keep image fallbacks CSP-compatible: inline onerror attributes are not
    // allowed by the nonce-based policy used by rendered public pages.
    document.addEventListener('error', function (event) {
      var image = event.target;
      if (!image || image.tagName !== 'IMG' || !image.dataset.fallbackSrc || image.dataset.fallbackUsed === 'true') return;
      image.dataset.fallbackUsed = 'true';
      image.src = image.dataset.fallbackSrc;
    }, true);

    // 1. Mobile Navigation Toggle
    var toggle = document.querySelector('.nav-toggle');
    var nav = document.getElementById('primary-nav');
    if (toggle && nav) {
      function toggleMenu(forceClose) {
        var isOpen = forceClose ? false : toggle.getAttribute('aria-expanded') !== 'true';
        toggle.setAttribute('aria-expanded', String(isOpen));
        toggle.setAttribute('aria-label', isOpen ? 'Menüyü kapat' : 'Menüyü aç');
        nav.classList.toggle('is-open', isOpen);
      }

      toggle.addEventListener('click', function (e) {
        e.stopPropagation();
        toggleMenu();
      });

      document.addEventListener('click', function (e) {
        if (toggle.getAttribute('aria-expanded') === 'true' && !nav.contains(e.target) && !toggle.contains(e.target)) {
          toggleMenu(true);
        }
      });

      document.addEventListener('keydown', function (e) {
        if (e.key === 'Escape' && toggle.getAttribute('aria-expanded') === 'true') {
          toggleMenu(true);
          toggle.focus();
        }
      });
    }

    // 2. Dynamic Header Elevation on Scroll
    var header = document.querySelector('.news-header');
    var backToTopBtn = document.getElementById('hydro-back-to-top');
    if (header || backToTopBtn) {
      function handleScroll() {
        var currentScroll = window.scrollY;
        if (header) {
          if (currentScroll > 15) {
            header.classList.add('is-scrolled');
          } else {
            header.classList.remove('is-scrolled');
          }
        }
        if (backToTopBtn) {
          if (currentScroll > 350) {
            backToTopBtn.classList.add('is-visible');
          } else {
            backToTopBtn.classList.remove('is-visible');
          }
        }
      }
      window.addEventListener('scroll', handleScroll, { passive: true });
      handleScroll();
    }

    if (backToTopBtn) {
      backToTopBtn.addEventListener('click', function () {
        window.scrollTo({ top: 0, behavior: 'smooth' });
      });
    }

    // 3. Scroll Reveal Intersection Observer
    var reveals = document.querySelectorAll('.theme-corporate-hero, .theme-corporate-testimonials, .theme-corporate-faq, .theme-corporate-pricing, .theme-corporate-cta, .testimonial-card, .pricing-card, .faq-item, .product-card, .news-card, .news-hero-banner, .news-article-card, .showcase-card, .stat-pill');
    if ('IntersectionObserver' in window && reveals.length > 0) {
      var observer = new IntersectionObserver(function (entries) {
        entries.forEach(function (entry) {
          if (entry.isIntersecting) {
            entry.target.classList.add('is-revealed');
            observer.unobserve(entry.target);
          }
        });
      }, {
        threshold: 0.08,
        rootMargin: '0px 0px -30px 0px'
      });

      reveals.forEach(function (el, index) {
        el.classList.add('reveal-item');
        if (el.classList.contains('testimonial-card') || el.classList.contains('pricing-card') || el.classList.contains('product-card') || el.classList.contains('news-card') || el.classList.contains('stat-pill')) {
          var delayIndex = (index % 4) + 1;
          el.classList.add('reveal-delay-' + delayIndex);
        }
        observer.observe(el);
      });
    } else {
      reveals.forEach(function (el) {
        el.classList.add('is-revealed');
      });
    }

    // 4. Animated Number Counters
    var counters = document.querySelectorAll('[data-counter]');
    if (counters.length > 0 && 'IntersectionObserver' in window) {
      var countObserver = new IntersectionObserver(function (entries) {
        entries.forEach(function (entry) {
          if (entry.isIntersecting) {
            var target = entry.target;
            var finalVal = parseFloat(target.getAttribute('data-counter')) || 0;
            var prefix = target.getAttribute('data-counter-prefix') || '';
            var suffix = target.getAttribute('data-counter-suffix') || '';
            var duration = parseInt(target.getAttribute('data-counter-duration'), 10) || 1200;
            var startTime = null;

            function animateCount(timestamp) {
              if (!startTime) startTime = timestamp;
              var progress = Math.min((timestamp - startTime) / duration, 1);
              // Easing out cubic
              var easeProgress = 1 - Math.pow(1 - progress, 3);
              var current = Math.floor(easeProgress * finalVal);
              target.textContent = prefix + current.toLocaleString('tr-TR') + suffix;
              if (progress < 1) {
                requestAnimationFrame(animateCount);
              } else {
                target.textContent = prefix + finalVal.toLocaleString('tr-TR') + suffix;
              }
            }
            requestAnimationFrame(animateCount);
            countObserver.unobserve(target);
          }
        });
      }, { threshold: 0.2 });

      counters.forEach(function (counter) {
        countObserver.observe(counter);
      });
    }

    // 5. Copy to Clipboard Utility
    document.addEventListener('click', function (e) {
      var copyBtn = e.target.closest('[data-copy], [data-copy-url]');
      if (!copyBtn) return;
      var textToCopy = '';
      if (copyBtn.hasAttribute('data-copy-url')) {
        textToCopy = window.location.href;
      } else {
        var copyTarget = copyBtn.getAttribute('data-copy');
        var targetEl = copyTarget ? document.querySelector(copyTarget) : null;
        textToCopy = targetEl ? (targetEl.innerText || targetEl.value) : copyTarget;
      }
      if (!textToCopy) return;

      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(textToCopy).then(function () {
          if (window.HydroUI && window.HydroUI.toast) {
            window.HydroUI.toast('Panoya kopyalandı', 'success');
          }
        }).catch(function () {
          fallbackCopy(textToCopy);
        });
      } else {
        fallbackCopy(textToCopy);
      }

      function fallbackCopy(val) {
        var tempInput = document.createElement('textarea');
        tempInput.value = val;
        tempInput.style.position = 'fixed';
        tempInput.style.opacity = '0';
        document.body.appendChild(tempInput);
        tempInput.select();
        try {
          document.execCommand('copy');
          if (window.HydroUI && window.HydroUI.toast) {
            window.HydroUI.toast('Panoya kopyalandı', 'success');
          }
        } catch (err) {}
        document.body.removeChild(tempInput);
      }
    });

    // 6. Interactive Tab Switcher
    document.addEventListener('click', function (e) {
      var tabBtn = e.target.closest('[data-tab-target]');
      if (!tabBtn) return;
      var targetId = tabBtn.getAttribute('data-tab-target');
      var container = tabBtn.closest('[data-tabs-container]') || document;

      var allButtons = container.querySelectorAll('[data-tab-target]');
      var allPanels = container.querySelectorAll('[data-tab-panel]');

      allButtons.forEach(function (btn) {
        var isActive = btn === tabBtn;
        btn.classList.toggle('is-active', isActive);
        btn.setAttribute('aria-selected', String(isActive));
      });

      allPanels.forEach(function (panel) {
        var isMatch = panel.getAttribute('data-tab-panel') === targetId;
        panel.classList.toggle('is-active', isMatch);
        panel.hidden = !isMatch;
      });
    });

    // 7. Quick Search Palette (Cmd+K / Ctrl+K)
    var searchModal = document.getElementById('hydro-quick-search');
    var searchTrigger = document.querySelector('[data-open-search]');
    var searchReturnFocus = null;

    function openSearch() {
      if (!searchModal) return;
      searchReturnFocus = document.activeElement;
      searchModal.classList.add('is-open');
      searchModal.setAttribute('aria-hidden', 'false');
      var input = searchModal.querySelector('input[type="search"], input[type="text"]');
      if (input) {
        input.value = '';
        setTimeout(function () { input.focus(); }, 50);
      }
      document.body.style.overflow = 'hidden';
    }

    function closeSearch() {
      if (!searchModal) return;
      searchModal.classList.remove('is-open');
      searchModal.setAttribute('aria-hidden', 'true');
      document.body.style.overflow = '';
      if (searchReturnFocus && typeof searchReturnFocus.focus === 'function') searchReturnFocus.focus();
    }

    if (searchTrigger) {
      searchTrigger.addEventListener('click', function (e) {
        e.preventDefault();
        openSearch();
      });
    }

    if (searchModal) {
      var closeBtn = searchModal.querySelector('[data-close-search]');
      if (closeBtn) closeBtn.addEventListener('click', closeSearch);

      searchModal.addEventListener('click', function (e) {
        if (e.target === searchModal) closeSearch();
      });

      searchModal.addEventListener('keydown', function (e) {
        if (e.key !== 'Tab') return;
        var focusable = searchModal.querySelectorAll('a[href], button:not([disabled]), input:not([disabled])');
        if (!focusable.length) return;
        var first = focusable[0];
        var last = focusable[focusable.length - 1];
        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault();
          last.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      });

      var searchInput = searchModal.querySelector('.quick-search-input');
      var searchItems = searchModal.querySelectorAll('.quick-search-item');
      if (searchInput && searchItems.length > 0) {
        searchInput.addEventListener('input', function () {
          var filter = searchInput.value.toLowerCase().trim();
          searchItems.forEach(function (item) {
            var text = item.textContent.toLowerCase();
            item.style.display = text.indexOf(filter) !== -1 ? '' : 'none';
          });
        });
      }
    }

    document.addEventListener('keydown', function (e) {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        if (searchModal && searchModal.classList.contains('is-open')) {
          closeSearch();
        } else {
          openSearch();
        }
      } else if (e.key === 'Escape') {
        if (searchModal && searchModal.classList.contains('is-open')) {
          closeSearch();
        }
      }
    });
    // Prevent accidental duplicate submissions while preserving native forms.
    document.addEventListener('submit', function (e) {
      var form = e.target;
      if (!(form instanceof HTMLFormElement) || form.dataset.submitting === 'true') return;
      form.dataset.submitting = 'true';
      Array.prototype.forEach.call(form.querySelectorAll('button[type="submit"], input[type="submit"]'), function (button) {
        button.disabled = true;
        button.setAttribute('aria-busy', 'true');
      });
    });
  }

  // ==========================================================================
  // HydroUI Public API
  // ==========================================================================
  var HydroUI = {
    toast: function (message, type, duration) {
      type = type || 'info';
      duration = duration || 3000;
      var container = document.getElementById('hydro-toast-container');
      if (!container) {
        container = document.createElement('div');
        container.id = 'hydro-toast-container';
        container.className = 'hydro-toast-container';
        container.setAttribute('aria-live', 'polite');
        container.setAttribute('aria-atomic', 'false');
        document.body.appendChild(container);
      }

      var toast = document.createElement('div');
      toast.className = 'hydro-toast hydro-toast-' + type;

      var icon = 'ℹ️';
      if (type === 'success') icon = '✓';
      if (type === 'error') icon = '✕';

      var iconEl = document.createElement('span');
      iconEl.className = 'hydro-toast-icon';
      iconEl.setAttribute('aria-hidden', 'true');
      iconEl.textContent = icon;
      var messageEl = document.createElement('span');
      messageEl.className = 'hydro-toast-msg';
      messageEl.textContent = String(message);
      toast.appendChild(iconEl);
      toast.appendChild(messageEl);
      container.appendChild(toast);

      // Trigger entrance
      requestAnimationFrame(function () {
        toast.classList.add('is-visible');
      });

      setTimeout(function () {
        toast.classList.remove('is-visible');
        setTimeout(function () {
          if (toast.parentNode) {
            toast.parentNode.removeChild(toast);
          }
        }, 300);
      }, duration);
    }
  };

  window.HydroUI = HydroUI;

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initUX);
  } else {
    initUX();
  }
})();
